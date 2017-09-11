package wrpendpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
)

const DefaultTimeout = 30 * time.Second

// New constructs a go-kit endpoint for the given WRP service
func New(s Service) endpoint.Endpoint {
	return func(ctx context.Context, value interface{}) (interface{}, error) {
		return s.ServeWRP(value.(Request))
	}
}

// Timeout applies the given timeout to all WRP Requests.  The context's cancellation
// function is always called.
func Timeout(timeout time.Duration) endpoint.Middleware {
	if timeout < 1 {
		timeout = DefaultTimeout
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, value interface{}) (interface{}, error) {
			var (
				timeoutCtx, cancel = context.WithTimeout(ctx, timeout)
				request            = value.(*request)
			)

			// generally, use WithContext instead of setting the context so that
			// the Request is immutable.
			//
			// However, this is a special case: we know that the request is not in use
			// by any other goroutine at this point in its lifecycle.  So, this avoids
			// a needless object allocation.
			request.ctx = timeoutCtx

			defer cancel()
			return next(timeoutCtx, request)
		}
	}
}
