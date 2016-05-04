package context

import (
	"net/http"
)

// ContextHandlerFunc is a function type for HTTP handlers that take a WebPA Context object
type ContextHandlerFunc func(Context, http.ResponseWriter, *http.Request)

// NewContextHttpHandler creates a new http.HandlerFunc which creates a new WebPA Context
// with each request.  This function is the primary entrypoint to this package for client code.
func NewContextHttpHandler(logger Logger, contextHandler ContextHandlerFunc) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer RecoverError(logger, response)

		context, err := NewContext(logger, request)
		if err != nil {
			panic(err)
		}

		contextHandler(context, response, request)
	})
}
