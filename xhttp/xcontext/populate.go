package xcontext

import (
	"context"
	"net/http"
	"time"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// Populate accepts any number of go-kit request functions and returns an Alice-style constructor that
// uses the request functions to build a context.  The resulting context is then assocated with the request
// prior to the next http.Handler being invoked.
//
// This function mimics the behavior of go-kit's transport/http package without requiring and endpoint with
// encoding and decoding.
func Populate(timeout time.Duration, rf ...gokithttp.RequestFunc) func(http.Handler) http.Handler {
	if timeout > 0 || len(rf) > 0 {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				ctx := Context(response, request)
				for _, f := range rf {
					ctx = f(ctx, request)
				}

				if timeout > 0 {
					var cancel func()
					ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()
				}
				response, request = WithContext(response, request, ctx)
				next.ServeHTTP(response, request)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return next
	}
}
