package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

// LogHandler logs provides a handler middleware that logs the headers of a requests for a given errorCode.
func LogHandler(errorCode int, l log.Logger) (h func(http.Handler) http.Handler) {
	h = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if errorCode < 100 || errorCode > 511 {
				h.ServeHTTP(response, request)
			} else if errorCode == request.Response.StatusCode {
				b := new(bytes.Buffer)
				for _, v := range request.Header {
					fmt.Fprintf(b, "%s", v)
				}
				l.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "header requested with specified error code", b.String())
				h.ServeHTTP(response, request)
			} else {
				h.ServeHTTP(response, request)
			}
		})
	}
	return
}
