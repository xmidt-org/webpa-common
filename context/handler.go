package context

import (
	"net/http"
)

// ContextHandlerFunc is a function type for HTTP handlers that take a WebPA Context object
type ContextHandlerFunc func(Context, http.ResponseWriter, *http.Request)

// NewHttpHandler creates a new http.HandlerFunc which creates a new WebPA Context
// with each request.
func NewHttpHandler(logger Logger, contextHandler ContextHandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if err := WriteError(response, recovered); err != nil {
					logger.Error("Unable to write error to response: %v", err)
				}
			}
		}()

		context, err := NewContext(logger, request)
		if err != nil {
			panic(err)
		}

		contextHandler(context, response, request)
	})
}
