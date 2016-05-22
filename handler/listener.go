package handler

import (
	"golang.org/x/net/context"
	"net/http"
)

// RequestListener gets notified of certain high-level request events
type RequestListener interface {
	// RequestReceived is invoked anytime a handler receives a request
	RequestReceived(*http.Request)

	// RequestCompleted is invoked after the response has been written.
	// The first parameter is the response status code, e.g. http.StatusOk
	RequestCompleted(int, *http.Request)
}

// ListenableResponseWriter wraps a http.ResponseWriter and records the
// status code.
type listenableResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (writer *listenableResponseWriter) WriteHeader(statusCode int) {
	writer.ResponseWriter.WriteHeader(statusCode)
	writer.statusCode = statusCode
}

// Listen produces a ChainHandler that notifies the given list of listeners
// of request events.
func Listen(listeners ...RequestListener) ChainHandler {
	return ChainHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
		for _, listener := range listeners {
			listener.RequestReceived(request)
		}

		listenableResponse := &listenableResponseWriter{ResponseWriter: response}
		next.ServeHTTP(ctx, listenableResponse, request)

		statusCode := listenableResponse.statusCode
		if statusCode < 1 {
			statusCode = http.StatusOK
		}

		for _, listener := range listeners {
			listener.RequestCompleted(statusCode, request)
		}
	})
}
