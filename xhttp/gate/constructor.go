package gate

import (
	"net/http"
)

// constructor is a configurable Alice-style decorator for HTTP handlers that controls
// traffic based on the current state of a gate.
type constructor struct {
	g      Interface
	closed http.Handler
}

func (c *constructor) decorate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if c.g.Open() {
			//filter and see if request should go through
			if request.Body != nil {
				msg, err := RequestToWRP(request)

				if msg != nil && err != nil {
					if c.g.Filters().FilterRequest(*msg) {
						next.ServeHTTP(response, request)
					}
				} else {
					c.closed.ServeHTTP(response, request)
				}
			}
		} else {
			c.closed.ServeHTTP(response, request)
		}
	})
}

func defaultClosedHandler(response http.ResponseWriter, _ *http.Request) {
	response.WriteHeader(http.StatusServiceUnavailable)
}

// ConstructorOption configures a gate decorator
type ConstructorOption func(*constructor)

// WithClosedHandler configures an arbitrary http.Handler that will serve requests when a gate is closed.
// If the handler is nil, the internal default is used instead.
func WithClosedHandler(closed http.Handler) ConstructorOption {
	return func(c *constructor) {
		if closed != nil {
			c.closed = closed
		} else {
			c.closed = http.HandlerFunc(defaultClosedHandler)
		}
	}
}

// NewConstructor returns an Alice-style constructor which decorates HTTP handlers with gating logic.  If supplied, the closed
// handler is invoked instead of the decorated handler whenever the gate is closed.  The closed handler may be nil, in which
// case a default is used that returns http.StatusServiceUnavailable.
//
// If g is nil, this function panics.
func NewConstructor(g Interface, options ...ConstructorOption) func(http.Handler) http.Handler {
	if g == nil {
		panic("A gate is required")
	}

	c := &constructor{
		g:      g,
		closed: http.HandlerFunc(defaultClosedHandler),
	}

	for _, o := range options {
		o(c)
	}

	return c.decorate
}
