package xfilter

import (
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// Option is a configuration option for a filter constructor
type Option func(*constructor)

func WithFilters(f ...Interface) Option {
	return func(c *constructor) {
		c.filters = append(c.filters, f...)
	}
}

func WithErrorEncoder(ee gokithttp.ErrorEncoder) Option {
	return func(c *constructor) {
		if ee != nil {
			c.errorEncoder = ee
		} else {
			c.errorEncoder = gokithttp.DefaultErrorEncoder
		}
	}
}

// NewConstructor returns an Alice-style decorator that filters requests
// sent to the decorated handler.  If no filters are configured, the returned
// constructor simply returns the handler unmodified.
func NewConstructor(o ...Option) func(http.Handler) http.Handler {
	c := &constructor{
		errorEncoder: gokithttp.DefaultErrorEncoder,
	}

	for _, f := range o {
		f(c)
	}

	return c.decorate
}

// constructor is the internal contextual type for decoration
type constructor struct {
	errorEncoder gokithttp.ErrorEncoder
	filters      []Interface
}

func (c constructor) decorate(next http.Handler) http.Handler {
	if len(c.filters) > 0 {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			for _, f := range c.filters {
				if err := f.Allow(request); err != nil {
					c.errorEncoder(request.Context(), err, response)
					return
				}
			}

			next.ServeHTTP(response, request)
		})
	}

	return next
}
