package health

import (
	"net/http"
)

// Wrap returns a *health.ResponseWriter which wraps the given
// http.ResponseWriter
func Wrap(delegate http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: delegate,
	}
}

// ResponseWriter is a wrapper type for an http.ResponseWriter that exposes the status code.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *ResponseWriter) StatusCode() int {
	return r.statusCode
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
