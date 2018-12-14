package bookkeeping

import (
	"bytes"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp/xcontext"
	"net/http"
	"time"
)

type CapturedResponse struct {
	Code    int
	Payload []byte
	Header  http.Header
}

// RequestFunc takes the request and returns key value pairs from the request
type RequestFunc func(request *http.Request) []interface{}

// ResponseFunc takes the ResponseWriter and returns key value pairs from the request
type ResponseFunc func(response CapturedResponse) []interface{}

// handler is for logging of the request and response
type handler struct {
	next   http.Handler
	before []RequestFunc
	after  []ResponseFunc
}

// Option provides a single configuration option for a bookkeeping handler
type Option func(h *handler)

func New(options ...Option) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("next can't be nil")
		}
		h := &handler{
			next: next,
		}
		for _, o := range options {
			o(h)
		}
		return h
	}
}

// WithRequests take a one or many RequestFuncs to build an Option for logging key value pairs from the
// RequestFun
func WithRequests(requestFuncs ...RequestFunc) Option {
	return func(h *handler) {
		h.before = append(h.before, requestFuncs...)
	}
}

// WithResponses take a one or many ResponseFunc to build an Option for logging key value pairs from the
// ResponseFunc
func WithResponses(responseFuncs ...ResponseFunc) Option {
	return func(h *handler) {
		h.after = append(h.after, responseFuncs...)
	}
}

// ServeHTTP handles the bookkeeping given
func (h *handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	kv := []interface{}{logging.MessageKey(), "Bookkeeping transactor"}
	for _, before := range h.before {
		kv = append(kv, before(request)...)
	}
	responseWriter, request = xcontext.WithContext(responseWriter, request, request.Context())
	w := &writerInterceptor{ResponseWriter: responseWriter, ContextAware: responseWriter.(xcontext.ContextAware)}

	start := time.Now()
	defer func() {
		ctx := xcontext.Context(responseWriter, request)
		duration := time.Since(start)

		kv = append(kv, "duration", duration)

		response := CapturedResponse{
			Code:    w.code,
			Payload: w.buffer.Bytes(),
			Header:  w.Header(),
		}

		for _, after := range h.after {
			kv = append(kv, after(response)...)
		}

		logging.Info(logging.GetLogger(ctx)).Log(kv...)
	}()

	h.next.ServeHTTP(w, request)
}

type writerInterceptor struct {
	xcontext.ContextAware
	http.ResponseWriter
	code   int
	buffer bytes.Buffer
}

func (w *writerInterceptor) Write(data []byte) (int, error) {
	w.buffer.Write(data)
	return w.ResponseWriter.Write(data)

}

func (w *writerInterceptor) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
