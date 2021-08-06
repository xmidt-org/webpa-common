package middleware

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

// loggable is the interface implemented by any message object which is associated with a go-kit Logger
type loggable interface {
	Logger() log.Logger
}

// Logging is a go-kit middleware that inserts any associated logger from requests into the context.
// Requests that do not provide a Logger() log.Logger method are simply ignored.
//
// This middleware is primarily useful because go-kit does not allow you to alter the context when requests
// are decoded.  That means that any contextual logger created when the request was decoded isn't visible
// in the context, unless something like this middleware is used.
func Logging(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, value interface{}) (interface{}, error) {
		if l, ok := value.(loggable); ok {
			return next(
				logging.WithLogger(ctx, l.Logger()),
				value,
			)
		}

		return next(ctx, value)
	}
}
