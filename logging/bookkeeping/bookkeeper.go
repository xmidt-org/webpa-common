package bookkeeping

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/justinas/alice"
	"net/http"
	"time"
)

// RequestFunc takes the request and returns key value pairs from the request
type RequestFunc func(request *http.Request) []interface{}

// ResponseFunc takes the ResponseWriter and returns key value pairs from the request
type ResponseFunc func(response http.ResponseWriter) []interface{}

// Handler is for logging of the request and response
type Handler struct {
	Logger log.Logger
	before []RequestFunc
	after  []ResponseFunc
}

// buildChain creates an alice.Chain to be built upon the handler.
func (h *Handler) buildChain() alice.Chain {
	return alice.New(func(next http.Handler) http.Handler {
		kv := make([]interface{}, 0)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			for _, before := range h.before {
				kv = append(kv, before(request)...)
			}

			start := time.Now()
			next.ServeHTTP(response, request)
			duration := time.Since(start)

			for _, after := range h.after {
				kv = append(kv, after(response)...)
			}
			kv = append(kv, "duration", duration)
			h.Logger.Log(kv...)
		})
	})
}

// Option provides a single configuration option for a bookkeeping Handler
type Option func(h *Handler)

// New creates a new Handler to then generate an alice.Chain for handling bookkeeping of the transaction
func New(logger log.Logger, options ...Option) alice.Chain {
	if logger == nil {
		logger = logging.DefaultLogger()
	}
	h := &Handler{
		Logger: log.With(logger, logging.MessageKey(), "bookkeeping logging"),
	}
	for _, o := range options {
		o(h)
	}
	return h.buildChain()
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
