package xhttp

import (
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/semaphore"
)

// Busy creates an Alice-style constructor that limits the number of HTTP transactions handled by decorated
// handlers.  The decorated handler blocks waiting on a semaphore until the request's context is canceled.
// If a transaction is not allowed to proceed, http.StatusServiceUnavailable.
func Busy(maxTransactions int) func(http.Handler) http.Handler {
	s := semaphore.New(maxTransactions)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			ctx := request.Context()

			if err := s.AcquireCtx(ctx); err != nil {
				logging.GetLogger(ctx).Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "server busy", logging.ErrorKey(), request.Context().Err())
				response.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			defer s.Release()
			next.ServeHTTP(response, request)
		})
	}
}
