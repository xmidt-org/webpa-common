package fanout

import "github.com/go-kit/kit/endpoint"

// Components holds the component endpoint objects which will be concurrently invoked by a fanout.
type Components map[string]endpoint.Endpoint

// Apply produces a new Components with each endpoint decorated by the given middleware.  To apply
// multiple middleware in one shot, pass the result of endpoint.Chain to this method.
func (c Components) Apply(m endpoint.Middleware) Components {
	decorated := make(Components, len(c))
	for k, v := range c {
		decorated[k] = m(v)
	}

	return decorated
}
