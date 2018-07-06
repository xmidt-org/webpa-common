package fanout

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/tracing/tracinghttp"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	gokithttp "github.com/go-kit/kit/transport/http"
)

var (
	errNoFanoutURLs  = errors.New("No fanout URLs")
	errBadTransactor = errors.New("Transactor did not conform to stdlib API")
)

// Option provides a single configuration option for a fanout Handler
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

// WithDecoder configures a custom decoder applied to the original request.  If the supplied decoder is nil,
// the DefaultDecoder is used.
func WithDecoder(d Decoder) Option {
	return func(h *Handler) {
		if d == nil {
			h.decoder = DefaultDecoder
		} else {
			h.decoder = d
		}
	}
}

// WithEncoder configures a custom encoder applied to the original request.  If the supplied
// encoder is nil, the DefaultEncoder is used.
func WithEncoder(e Encoder) Option {
	return func(h *Handler) {
		if e == nil {
			h.encoder = DefaultEncoder
		} else {
			h.encoder = e
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

// WithFanoutBefore adds zero or more fanout request functions that will tailor each fanout request.
//
// To apply one or more go-kit RequestFuncs, use WithClientBefore.
func WithFanoutBefore(before ...RequestFunc) Option {
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
				func(ctx context.Context, _, fanout *http.Request) (context.Context, error) {
					return rf(ctx, fanout), nil
				},
			)
		}
	}
}

// WithFanoutAfter adds zero or more response functions that are invoked to tailor the response
// when a successful (i.e. terminating) fanout response is received.
//
// To apply one or more go-kit ClientResponseFuncs, use WithClientAfter.
func WithFanoutAfter(after ...ResponseFunc) Option {
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
				func(ctx context.Context, fanout Result, original http.Header) context.Context {
					return rf(ctx, fanout.Response)
				},
			)
		}
	}
}

// WithConfiguration uses a set of (typically injected) fanout configuration options to configure a Handler.
// Use of this option will not override the configured Endpoints instance.
//
// One of the aspects tailored by the returned Option is the HTTP client transactor strategy.  The returned
// option will use the same transactor every time it is applied to a fanout.  If different transactors are
// desired, invoke this function for each fanout.  For example:
//
//    var cfg Configuration
//    option := WithConfiguration(cfg)
//    fanout1 := New(option)
//    fanout2 := New(option) // same transactor as fanout1
//    fanout3 := New(WithConfiguration(cfg)) // different transactor than fanout1
func WithConfiguration(c Configuration) Option {
	transactor := NewTransactor(c)
	return func(h *Handler) {
		WithTransactor(transactor)(h)

		authorization := c.authorization()
		if len(authorization) > 0 {
			WithClientBefore(gokithttp.SetRequestHeader("Authorization", "Basic "+authorization))(h)
		}
	}
}

// Handler is the http.Handler that fans out HTTP requests using the configured Endpoints strategy.
type Handler struct {
	endpoints       Endpoints
	errorEncoder    gokithttp.ErrorEncoder
	decoder         Decoder
	encoder         Encoder
	before          []RequestFunc
	after           []ResponseFunc
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
		decoder:         DefaultDecoder,
		encoder:         DefaultEncoder,
		shouldTerminate: DefaultShouldTerminate,
		transactor:      http.DefaultClient.Do,
	}

	for _, o := range options {
		o(h)
	}

	return h
}

// newFanoutRequests uses the Endpoints strategy and builds (1) HTTP request for each endpoint.  The configured
// RequestFunc options are used to build each request.  This method returns an error if no endpoints were returned
// by the strategy or if an error reading the original request body occurred.
func (h *Handler) newFanoutRequests(fanoutCtx context.Context, original *http.Request) ([]*http.Request, error) {
	urls, err := h.endpoints.FanoutURLs(original)
	if err != nil {
		return nil, err
	} else if len(urls) == 0 {
		return nil, errNoFanoutURLs
	}

	fanoutHeader := make(http.Header)
	ctx, fanoutBody, err := h.decoder(original.Context(), original, fanoutHeader)
	if err != nil {
		return nil, err
	}

	requests := make([]*http.Request, len(urls))
	for i := 0; i < len(urls); i++ {
		fanout := &http.Request{
			Method:     original.Method,
			URL:        urls[i],
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Host:       urls[i].Host,
		}

		fanout.Body, fanout.GetBody = xhttp.NewRewindBytes(fanoutBody)
		for k, values := range fanoutHeader {
			for _, v := range values {
				fanout.Header.Add(k, v)
			}
		}

		fanoutCtx := ctx
		var err error
		for _, rf := range h.before {
			fanoutCtx, err = rf(fanoutCtx, original, fanout)
			if err != nil {
				return nil, err
			}
		}

		requests[i] = fanout.WithContext(fanoutCtx)
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
		result.ContentType = result.Response.Header.Get("Content-Type")

		var err error
		if result.Body, err = ioutil.ReadAll(result.Response.Body); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error reading fanout response body", logging.ErrorKey(), err)
		}

		if err = result.Response.Body.Close(); err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error closing fanout response body", logging.ErrorKey(), err)
		}

	case result.Err != nil:
		result.Body = []byte(fmt.Sprintf("%s", result.Err))
		result.ContentType = "text/plain"

		if ue, ok := result.Err.(*url.Error); ok && ue.Err != nil {
			// unwrap the URL error
			result.Err = ue.Err
		}

		if result.Err == context.Canceled || result.Err == context.DeadlineExceeded {
			result.StatusCode = http.StatusGatewayTimeout
		} else {
			result.StatusCode = http.StatusServiceUnavailable
		}

	default:
		// this "should" never happen, but just in case set a known status code
		result.StatusCode = http.StatusInternalServerError
		result.Err = errBadTransactor
		result.Body = []byte(errBadTransactor.Error())
		result.ContentType = "test/plain"
	}

	result.Span = finisher(result.Err)
	results <- result
}

// finish takes a terminating fanout result and writes the appropriate information to the top-level response.  This method
// is only invoked when a particular fanout response terminates the fanout, i.e. is considered successful.
func (h *Handler) finish(logger log.Logger, response http.ResponseWriter, original *http.Request, fanout Result) {
	var (
		ctx    = fanout.Request.Context()
		header = response.Header()
	)

	for _, rf := range h.after {
		ctx = rf(ctx, fanout, header)
	}

	body, err := h.encoder(ctx, fanout, header)
	if err != nil {
		h.errorEncoder(ctx, err, response)
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error while encoding fanout response", logging.ErrorKey(), err)
		return
	}

	count, err := response.Write(body)
	logLevel := level.DebugValue()
	if err != nil {
		logLevel = level.ErrorValue()
	}

	logger.Log(level.Key(), logLevel, logging.MessageKey(), "wrote encoded fanout response", "count", count, logging.ErrorKey(), err)
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
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "fanout operation canceled or timed out", "statusCode", http.StatusGatewayTimeout, "url", original.URL, logging.ErrorKey(), fanoutCtx.Err())
			response.WriteHeader(http.StatusGatewayTimeout)
			return

		case r := <-results:
			tracinghttp.HeadersForSpans("", response.Header(), r.Span)
			logLevel := level.DebugValue()
			if r.Err != nil {
				logLevel = level.ErrorValue()
			}

			logger.Log(level.Key(), logLevel, logging.MessageKey(), "fanout request complete", "statusCode", r.StatusCode, "url", r.Request.URL, logging.ErrorKey(), r.Err)

			if h.shouldTerminate(r) {
				// this was a "success", so no reason to wait any longer
				h.finish(logger, response, original, r)
				return
			}

			if statusCode < r.StatusCode {
				statusCode = r.StatusCode
			}
		}
	}

	logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "all fanout requests failed", "statusCode", statusCode, "url", original.URL)
	response.WriteHeader(statusCode)
}
