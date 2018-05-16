package xhttp

import (
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log/level"
)

// Busy creates an Alice-style constructor that limits the number of HTTP transactions handled by decorated
// handlers.  The decorated handler blocks waiting on a semaphore until the request's context is canceled.
// If a transaction is not allowed to proceed, http.StatusServiceUnavailable.
func Busy(maxTransactions int) func(http.Handler) http.Handler {
	if maxTransactions < 1 {
		panic("maxTransactions must be positive")
	}

	var (
		semaphore = make(chan struct{}, maxTransactions)
		release   = func() {
			<-semaphore
		}
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			logger := logging.GetLogger(request.Context())
			select {
			case <-request.Context().Done():
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "server busy", logging.ErrorKey(), request.Context().Err())
				response.WriteHeader(http.StatusServiceUnavailable)

			case semaphore <- struct{}{}:
				defer release()
				next.ServeHTTP(response, request)
			}
		})
	}
}
