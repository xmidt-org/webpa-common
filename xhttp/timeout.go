package xhttp

import (
	"context"
	"net/http"
	"time"
)

// Timeout returns an Alice-style constructor that applies a timeout to all request contexts.
// If timeout is nonpositive, the returned constructor simply returns the next http.Handler undecorated.
//
// The returned constructor does not enforce the timeout in any way.  Decorated http.Handler code is responsible
// for timing out as appropriate.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if timeout < 1 {
			return next
		}

		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			ctx, cancel := context.WithTimeout(request.Context(), timeout)
			defer cancel()

			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}
