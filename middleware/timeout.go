package middleware

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
)

const DefaultTimeout = 30 * time.Second

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
