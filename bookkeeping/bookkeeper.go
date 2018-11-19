package bookkeeping

import (
	"bytes"
	"github.com/Comcast/webpa-common/logging"
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

// Handler is for logging of the request and response
type Handler struct {
	next   http.Handler
	before []RequestFunc
	after  []ResponseFunc
}

// Option provides a single configuration option for a bookkeeping Handler
type Option func(h *Handler)

func New(options ...Option) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil{
			panic("next can't be nil")
		}
		h := &Handler{
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
	return func(h *Handler) {
		h.before = append(h.before, requestFuncs...)
	}
}

// WithResponses take a one or many ResponseFunc to build an Option for logging key value pairs from the
// ResponseFunc
func WithResponses(responseFuncs ...ResponseFunc) Option {
	return func(h *Handler) {
		h.after = append(h.after, responseFuncs...)
	}
}

// ServeHTTP handles the bookkeeping given
func (h *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	kv := []interface{}{logging.MessageKey(), "Bookkeeping transactor"}
	for _, before := range h.before {
		kv = append(kv, before(request)...)
	}

	w := &writerInterceptor{ResponseWriter: responseWriter}

	start := time.Now()
	h.next.ServeHTTP(w, request)
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
	logging.GetLogger(request.Context()).Log(kv...)
}

type writerInterceptor struct {
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
