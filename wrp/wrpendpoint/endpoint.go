package wrpendpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
)

const DefaultTimeout = 30 * time.Second

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

// Timeout applies the given timeout to all WRP Requests.  The context's cancellation
// function is always called.
func Timeout(timeout time.Duration) endpoint.Middleware {
	if timeout < 1 {
		timeout = DefaultTimeout
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, value interface{}) (interface{}, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			return next(timeoutCtx, value)
		}
	}
}
