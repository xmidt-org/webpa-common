package wrpendpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// New constructs a go-kit endpoint for the given WRP service.  This endpoint enforces
// the constraint that ctx must be the context associated with the Request.
func New(s Service) endpoint.Endpoint {
	return func(ctx context.Context, value interface{}) (interface{}, error) {
		return s.ServeWRP(ctx, value.(Request))
	}
}

// Wrap does the opposite of New: it takes a go-kit endpoint and returns a Service
// that invokes it.
func Wrap(e endpoint.Endpoint) Service {
	return ServiceFunc(func(ctx context.Context, request Request) (Response, error) {
		response, err := e(ctx, request)
		return response.(Response), err
	})
}
