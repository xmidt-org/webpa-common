package gate

import (
	"net/http"
)

func defaultClosedHandler(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusServiceUnavailable)
}

// NewConstructor returns an Alice-style constructor which decorates HTTP handlers with gating logic.  If supplied, the closed
// handler is invoked instead of the decorated handler whenever the gate is closed.  The closed handler may be nil, in which
// case a default is used that returns http.StatusServiceUnavailable.
//
// If g is nil, this function panics.
func NewConstructor(g Interface, closed http.Handler) func(http.Handler) http.Handler {
	if g == nil {
		panic("A gate is required")
	}

	if closed == nil {
		closed = http.HandlerFunc(defaultClosedHandler)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if g.IsOpen() {
				next.ServeHTTP(response, request)
			} else {
				closed.ServeHTTP(response, request)
			}
		})
	}
}
