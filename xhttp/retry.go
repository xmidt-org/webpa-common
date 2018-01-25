package xhttp

import (
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// temporaryError is the expected interface for a (possibly) temporary error
type temporaryError interface {
	Temporary() bool
}

// ShouldRetryFunc is a predicate for determining if the error returned from an HTTP transaction
// should be retried.
type ShouldRetryFunc func(error) bool

// DefaultShouldRetry is the default retry predicate.  It returns true if and only if err exposes a Temporary() bool
// method and that method returns true.
func DefaultShouldRetry(err error) bool {
	if temp, ok := err.(temporaryError); ok {
		return temp.Temporary()
	}

	return false
}

// RetryTransactor returns an HTTP transactor function, of the same signature as http.Client.Do, that
// retries a certain number of times.
//
// If retryCount is nonpositive, next is returned undecorated.
func RetryTransactor(logger log.Logger, retryCount int, shouldRetry ShouldRetryFunc, next func(*http.Request) (*http.Response, error)) func(*http.Request) (*http.Response, error) {
	if retryCount < 1 {
		return next
	}

	if shouldRetry == nil {
		shouldRetry = DefaultShouldRetry
	}

	attempts := retryCount + 1
	return func(request *http.Request) (*http.Response, error) {
		var (
			response *http.Response
			err      error
		)

		for i := 0; i < attempts; i++ {
			response, err = next(request)
			if err != nil && shouldRetry(err) {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "retrying HTTP transaction", "url", request.URL.String(), "error", err, "attempt", i+1)
				continue
			}

			break
		}

		if err != nil {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "All HTTP transaction retries failed", "url", request.URL.String(), "error", err, "attempts", attempts)
		}

		return response, err
	}
}
