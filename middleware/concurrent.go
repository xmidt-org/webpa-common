package middleware

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// Concurrent produces a middleware that allows only a set number of concurrent calls via
// a semaphore implemented as a buffered channel.  The context is used for cancellation,
// and if the context is cancelled then timeoutError is returned if it is not nil, ctx.Err() otherwise.
func Concurrent(concurrency int, timeoutError error) endpoint.Middleware {
	semaphore := make(chan struct{}, concurrency)
	for r := 0; r < concurrency; r++ {
		semaphore <- struct{}{}
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, value interface{}) (interface{}, error) {
			select {
			case <-ctx.Done():
				if timeoutError != nil {
					return nil, timeoutError
				} else {
					return nil, ctx.Err()
				}

			case <-semaphore:
			}

			defer func() {
				semaphore <- struct{}{}
			}()

			return next(ctx, value)
		}
	}
}
