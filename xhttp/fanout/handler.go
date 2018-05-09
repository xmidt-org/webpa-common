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
	errBadTransactor     = errors.New("Transactor did not conform to stdlib API")
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

// WithFanoutAfter adds zero or more response functions that are invoked to tailor the response
// when a successful (i.e. terminating) fanout response is received.
func WithFanoutAfter(after ...FanoutResponseFunc) Option {
	return func(h *Handler) {
		h.after = append(h.after, after...)
	}
}

// WithClientAfter allows zero or more go-kit ClientResponseFuncs to be used as fanout after functions.
func WithClientAfter(after ...gokithttp.ClientResponseFunc) Option {
	return func(h *Handler) {
		for _, rf := range after {
			h.after = append(
				h.after,
				func(ctx context.Context, response http.ResponseWriter, result Result) context.Context {
					return rf(ctx, result.Response)
				},
			)
		}
	}
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
//
// By default, all fanout requests have the same HTTP method as the original request, but no body is set..  Clients must use the OriginalBody
// strategy to set the original request's body on each fanout request.
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
func (h *Handler) execute(logger log.Logger, spanner tracing.Spanner, results chan<- Result, request *http.Request) {
	var (
		finisher = spanner.Start(request.URL.String())
		result   = Result{
			Request: request,
		}
	)

	result.Response, result.Err = h.transactor(request)
	switch {
	case result.Response != nil:
		result.StatusCode = result.Response.StatusCode
		var err error
		if result.Body, err = ioutil.ReadAll(result.Response.Body); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error reading fanout response body", logging.ErrorKey(), err)
		}

		if err = result.Response.Body.Close(); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error closing fanout response body", logging.ErrorKey(), err)
		}

	case result.Err != nil:
		if result.Err == context.Canceled || result.Err == context.DeadlineExceeded {
			result.StatusCode = http.StatusGatewayTimeout
		} else {
			result.StatusCode = http.StatusServiceUnavailable
		}

	default:
		// this "should" never happen, but just in case set a known status code
		result.StatusCode = http.StatusInternalServerError
		result.Err = errBadTransactor
	}

	result.Span = finisher(result.Err)
	results <- result
}

// finish takes a terminating fanout result and writes the appropriate information to the top-level response.  This method
// is only invoked when a particular fanout response terminates the fanout, i.e. is considered successful.
func (h *Handler) finish(logger log.Logger, response http.ResponseWriter, result Result) {
	ctx := result.Request.Context()
	for _, rf := range h.after {
		// NOTE: we don't use the context for anything here,
		// but to preserve go-kit semantics we pass it to each after function
		ctx = rf(ctx, response, result)
	}

	response.Header().Set("Content-Type", result.Response.Header.Get("Content-Type"))
	response.WriteHeader(result.StatusCode)
	if count, err := response.Write(result.Body); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error writing response body", logging.ErrorKey(), err)
	} else {
		logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "wrote fanout response", "bytes", count)
	}
}

func (h *Handler) ServeHTTP(response http.ResponseWriter, original *http.Request) {
	var (
		fanoutCtx     = original.Context()
		logger        = logging.GetLogger(fanoutCtx)
		requests, err = h.newFanoutRequests(fanoutCtx, original)
	)

	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to create fanout", logging.ErrorKey(), err)
		h.errorEncoder(fanoutCtx, err, response)
		return
	}

	var (
		spanner = tracing.NewSpanner()
		results = make(chan Result, len(requests))
	)

	for _, r := range requests {
		go h.execute(logger, spanner, results, r)
	}

	statusCode := 0
	for i := 0; i < len(requests); i++ {
		select {
		case <-fanoutCtx.Done():
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "fanout operation canceled or timed out", logging.ErrorKey(), fanoutCtx.Err())
			response.WriteHeader(http.StatusGatewayTimeout)
			return

		case r := <-results:
			tracinghttp.HeadersForSpans("", response.Header(), r.Span)
			logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "fanout operation complete", "statusCode", r.StatusCode, "url", r.Request.URL)

			if h.shouldTerminate(r) {
				// this was a "success", so no reason to wait any longer
				h.finish(logger, response, r)
				return
			}

			if statusCode < r.StatusCode {
				statusCode = r.StatusCode
			}
		}
	}

	response.WriteHeader(statusCode)
}
