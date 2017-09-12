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
		request := value.(*request)
		request.ctx = ctx

		return s.ServeWRP(request)
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

			request.ctx = timeoutCtx
			defer cancel()
			return next(timeoutCtx, request)
		}
	}
}
