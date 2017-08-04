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

// CloseNotify delegates to the wrapped ResponseWriter, panicking if the delegate does
// not implement http.CloseNotifier.
func (r *ResponseWriter) CloseNotify() <-chan bool {
	if closeNotifier, ok := r.ResponseWriter.(http.CloseNotifier); ok {
		return closeNotifier.CloseNotify()
	}

	// TODO: Not quite sure what the best thing to do here is.  At least with
	// a panic, we'll catch it later and can decide what to do.
	panic(errors.New("Wrapped response does not implement http.CloseNotifier"))
}

// Hijack delegates to the wrapped ResponseWriter, returning an error if the delegate does
// not implement http.Hijacker.
func (r *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}

	return nil, nil, errors.New("Wrapped response does not implement http.Hijacker")
}

// Flush delegates to the wrapped ResponseWriter.  If the delegate ResponseWriter does not
// implement http.Flusher, this method does nothing.
func (r *ResponseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Push delegates to the wrapper ResponseWriter, returning an error if the delegate does
// not implement http.Pusher.
func (r *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return errors.New("Wrapped response does not implement http.Pusher")
}
