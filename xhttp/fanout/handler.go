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

// Options is a configuration option for a fanout Handler
type Option func(*Handler)

// WithShouldTerminate configures a custom termination predicate for the fanout.  If terminate
// is nil, DefaultShouldTerminate is used.
func WithShouldTerminate(terminate ShouldTerminateFunc) Option {
	return func(h *Handler) {
		if terminate != nil {
			h.shouldTerminate = terminate
		} else {
			h.shouldTerminate = DefaultShouldTerminate
		}
	}
}

// WithErrorEncoder configures a custom error encoder for errors that occur during fanout setup.
// If encoder is nil, go-kit's DefaultErrorEncoder is used.
func WithErrorEncoder(encoder gokithttp.ErrorEncoder) Option {
	return func(h *Handler) {
		if encoder != nil {
			h.errorEncoder = encoder
		} else {
			h.errorEncoder = gokithttp.DefaultErrorEncoder
		}
	}
}

// WithTransactor configures a custom HTTP client transaction function.  If transactor is nil,
// http.DefaultClient.Do is used as the transactor.
func WithTransactor(transactor func(*http.Request) (*http.Response, error)) Option {
	return func(h *Handler) {
		if transactor != nil {
			h.transactor = transactor
		} else {
			h.transactor = http.DefaultClient.Do
		}
	}
}

// WithFanoutBefore adds zero or more request functions that will tailor each fanout request.
func WithFanoutBefore(before ...FanoutRequestFunc) Option {
	return func(h *Handler) {
		h.before = append(h.before, before...)
	}
}

// WithClientBefore adds zero or more go-kit RequestFunc functions that will be applied to
// each fanout request.
func WithClientBefore(before ...gokithttp.RequestFunc) Option {
	return func(h *Handler) {
		for _, rf := range before {
			h.before = append(
				h.before,
				func(ctx context.Context, _, fanout *http.Request, _ []byte) context.Context {
					return rf(ctx, fanout)
				},
			)
		}
	}
}

// fanoutResult is the result from a single fanout HTTP transaction
type fanoutResult struct {
	statusCode int
	response   *http.Response
	body       []byte
	err        error
	span       tracing.Span
}

// Handler is the http.Handler that fans out HTTP requests using the configured Endpoints strategy.
type Handler struct {
	endpoints       Endpoints
	errorEncoder    gokithttp.ErrorEncoder
	before          []FanoutRequestFunc
	after           []FanoutResponseFunc
	shouldTerminate ShouldTerminateFunc
	transactor      func(*http.Request) (*http.Response, error)
}

// New creates a fanout Handler.  The Endpoints strategy is required, and this constructor function will
// panic if it is nil.
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

// newFanoutRequests uses the Endpoints strategy and builds (1) HTTP request for each endpoint.  The configured
// FanoutRequestFunc options are used to build each request.  This method returns an error if no endpoints were returned
// by the strategy or if an error reading the original request body occurred.
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

// execute performs a single fanout HTTP transaction and sends the result on a channel.  This method is invoked
// as a goroutine.  It takes care of draining the fanout's response prior to returning.
func (h *Handler) execute(logger log.Logger, spanner tracing.Spanner, results chan<- fanoutResult, request *http.Request) {
	var (
		finisher = spanner.Start(request.URL.String())
		result   fanoutResult
	)

	result.response, result.err = h.transactor(request)
	switch {
	case result.response != nil:
		result.statusCode = result.response.StatusCode
		var err error
		if result.body, err = ioutil.ReadAll(result.response.Body); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error reading fanout response body", logging.ErrorKey(), err)
		}

		if err = result.response.Body.Close(); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error closing fanout response body", logging.ErrorKey(), err)
		}

	case result.err != nil:
		if result.err == context.Canceled || result.err == context.DeadlineExceeded {
			result.statusCode = http.StatusGatewayTimeout
		} else {
			result.statusCode = http.StatusServiceUnavailable
		}
	}

	result.span = finisher(result.err)
	results <- result
}

// finish takes a terminating fanout result and writes the appropriate information to the top-level response.  This method
// is only invoked when a particular fanout response terminates the fanout, i.e. is considered successful.
func (h *Handler) finish(logger log.Logger, ctx context.Context, response http.ResponseWriter, result fanoutResult) {
	if result.response != nil {
		// if there was a response, use the original request's context, as it may have extra information
		ctx = result.response.Request.Context()
	}

	for _, rf := range h.after {
		ctx = rf(ctx, response, result.response, result.err)
	}

	response.Header().Set("Content-Type", result.response.Header.Get("Content-Type"))
	response.WriteHeader(result.statusCode)
	if _, err := response.Write(result.body); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error writing response body", logging.ErrorKey(), err)
	}
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

	statusCode := 0
	for i := 0; i < len(requests); i++ {
		select {
		case <-fanoutCtx.Done():
			response.WriteHeader(http.StatusGatewayTimeout)
			return

		case r := <-results:
			tracinghttp.HeadersForSpans("", response.Header(), r.span)

			if h.shouldTerminate(r.response, r.err) {
				// this was a "success", so no reason to wait any longer
				h.finish(logger, fanoutCtx, response, r)
				return
			}

			if statusCode < r.statusCode {
				statusCode = r.statusCode
			}
		}
	}

	response.WriteHeader(statusCode)
}
