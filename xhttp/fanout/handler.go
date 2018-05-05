package fanout

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/tracing/tracinghttp"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	gokithttp "github.com/go-kit/kit/transport/http"
)

var (
	errNoFanoutEndpoints = errors.New("No fanout endpoints")
)

type Option func(*Handler)

func WithShouldTerminate(terminate ShouldTerminateFunc) Option {
	return func(h *Handler) {
		if terminate != nil {
			h.shouldTerminate = terminate
		} else {
			h.shouldTerminate = DefaultShouldTerminate
		}
	}
}

func WithErrorEncoder(encoder gokithttp.ErrorEncoder) Option {
	return func(h *Handler) {
		if encoder != nil {
			h.errorEncoder = encoder
		} else {
			h.errorEncoder = gokithttp.DefaultErrorEncoder
		}
	}
}

func WithTransactor(transactor func(*http.Request) (*http.Response, error)) Option {
	return func(h *Handler) {
		if transactor != nil {
			h.transactor = transactor
		} else {
			h.transactor = http.DefaultClient.Do
		}
	}
}

func WithFanoutBefore(before ...FanoutRequestFunc) Option {
	return func(h *Handler) {
		h.before = append(h.before, before...)
	}
}

type fanoutResult struct {
	response *http.Response
	body     []byte
	err      error
	span     tracing.Span
}

type Handler struct {
	endpoints       Endpoints
	errorEncoder    gokithttp.ErrorEncoder
	before          []FanoutRequestFunc
	shouldTerminate ShouldTerminateFunc
	transactor      func(*http.Request) (*http.Response, error)
}

func New(e Endpoints, options ...Option) *Handler {
	if e == nil {
		panic("An Endpoints strategy is required")
	}

	h := &Handler{
		endpoints:       e,
		errorEncoder:    gokithttp.DefaultErrorEncoder,
		shouldTerminate: DefaultShouldTerminate,
		transactor:      http.DefaultClient.Do,
	}

	for _, o := range options {
		o(h)
	}

	return h
}

func (h *Handler) newFanoutRequests(fanoutCtx context.Context, original *http.Request) ([]*http.Request, error) {
	body, err := ioutil.ReadAll(original.Body)
	if err != nil {
		return nil, err
	}

	endpoints, err := h.endpoints.NewEndpoints(original)
	if err != nil {
		return nil, err
	} else if len(endpoints) == 0 {
		return nil, errNoFanoutEndpoints
	}

	requests := make([]*http.Request, len(endpoints))
	for i := 0; i < len(endpoints); i++ {
		fanout := &http.Request{
			Method:     original.Method,
			URL:        endpoints[i],
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Host:       endpoints[i].Host,
		}

		endpointCtx := fanoutCtx
		for _, rf := range h.before {
			endpointCtx = rf(endpointCtx, original, fanout, body)
		}

		requests[i] = fanout.WithContext(endpointCtx)
	}

	return requests, nil
}

func (h *Handler) execute(logger log.Logger, spanner tracing.Spanner, results chan<- fanoutResult, request *http.Request) {
	var (
		finisher = spanner.Start(request.URL.String())
		result   fanoutResult
	)

	result.response, result.err = h.transactor(request)
	if result.response != nil && result.response.Body != nil {
		var err error
		if result.body, err = ioutil.ReadAll(result.response.Body); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error reading fanout response body", logging.ErrorKey(), err)
		}

		if err = result.response.Body.Close(); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error closing fanout response body", logging.ErrorKey(), err)
		}
	} else if result.err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error during fanout transaction", logging.ErrorKey(), result.err)
	}

	result.span = finisher(result.err)
	results <- result
}

func (h *Handler) ServeHTTP(response http.ResponseWriter, original *http.Request) {
	var (
		fanoutCtx     = original.Context()
		logger        = logging.GetLogger(fanoutCtx)
		requests, err = h.newFanoutRequests(fanoutCtx, original)
	)

	if err != nil {
		h.errorEncoder(fanoutCtx, err, response)
		return
	}

	var (
		spanner = tracing.NewSpanner()
		results = make(chan fanoutResult, len(requests))
	)

	for _, r := range requests {
		go h.execute(logger, spanner, results, r)
	}

	largestCode := 0
	for i := 0; i < len(requests); i++ {
		select {
		case <-fanoutCtx.Done():
			response.WriteHeader(http.StatusGatewayTimeout)
			return

		case r := <-results:
			tracinghttp.HeadersForSpans("", response.Header(), r.span)

			if h.shouldTerminate(r.response, r.err) {
				response.Header().Set("Content-Type", r.response.Header.Get("Content-Type"))
				response.WriteHeader(r.response.StatusCode)
				if _, err := response.Write(r.body); err != nil {
					logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error writing response body", logging.ErrorKey(), err)
				}

				return
			} else if r.err == context.Canceled || r.err == context.DeadlineExceeded || fanoutCtx.Err() != nil {
				// consider a context cancellation just like a gateway timeout
				if http.StatusGatewayTimeout > largestCode {
					largestCode = http.StatusGatewayTimeout
				}
			} else if r.response != nil && r.response.StatusCode > largestCode {
				largestCode = r.response.StatusCode
			}
		}
	}

	if largestCode > 0 {
		response.WriteHeader(largestCode)
	} else {
		// nothing completed with a status code
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}
