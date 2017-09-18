package middleware

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-kit/kit/endpoint"
)

// Busy produces a middleware that returns an error if the maximum number of
// clients is reached.  If busyError is specified, that is returned.  Otherwise,
// a default error is returned.
//
// If maxClients is nonpositive, this factory function panics
func Busy(maxClients int64, busyError error) endpoint.Middleware {
	if maxClients < 1 {
		panic("maxClients must be positive")
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		if busyError == nil {
			busyError = fmt.Errorf("Exceeded maximum number of clients: %d", maxClients)
		}

		var clientCounter int64

		return func(ctx context.Context, value interface{}) (interface{}, error) {
			defer atomic.AddInt64(&clientCounter, -1)

			if atomic.AddInt64(&clientCounter, 1) > maxClients {
				return nil, busyError
			}

			return next(ctx, value)
		}
	}
}
