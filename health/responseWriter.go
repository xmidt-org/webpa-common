package health

import (
	"bufio"
	"errors"
	"net"
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

// Hijack delegates to the wrapped ResponseWriter, returning an error if the delegate does
// not implement http.Hijacker.
func (r *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, errors.New("Wrapped response does not implement http.Hijacker")
}
