package xhttp

import (
	"net/http"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/semaphore"
	"go.uber.org/zap"
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
				sallust.Get(ctx).Error("server busy", zap.Error(request.Context().Err()))
				response.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			defer s.Release()
			next.ServeHTTP(response, request)
		})
	}
}
