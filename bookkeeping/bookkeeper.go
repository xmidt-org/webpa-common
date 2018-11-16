package bookkeeping

import (
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	"time"
)

// RequestFunc takes the request and returns key value pairs from the request
type RequestFunc func(request *http.Request) []interface{}

// ResponseFunc takes the ResponseWriter and returns key value pairs from the request
type ResponseFunc func(response *http.Response) []interface{}

// Handler is for logging of the request and response
type Handler struct {
	next   func(*http.Request) (*http.Response, error)
	before []RequestFunc
	after  []ResponseFunc
}

// transact is the HTTP transactor function that does the Bookkeeping
func (h *Handler) transact(request *http.Request) (*http.Response, error) {
	kv := []interface{}{logging.MessageKey(), "Bookkeeping transactor"}
	for _, before := range h.before {
		kv = append(kv, before(request)...)
	}

	start := time.Now()
	response, err := h.next(request)
	duration := time.Since(start)
	kv = append(kv, "duration", duration)

	if err != nil {
		kv = append(kv, logging.MessageKey(), err)
	}
	for _, after := range h.after {
		kv = append(kv, after(response)...)
	}

	logging.GetLogger(request.Context()).Log(kv...)
	return response, err
}

// Option provides a single configuration option for a bookkeeping Handler
type Option func(h *Handler)

// Transactor returns an HTTP transactor function used for Bookkeeping the request and response
func Transactor(next func(*http.Request) (*http.Response, error), options ...Option) func(*http.Request) (*http.Response, error) {
	h := &Handler{
		next: next,
	}
	for _, o := range options {
		o(h)
	}
	return h.transact
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
