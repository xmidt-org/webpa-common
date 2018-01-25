package xhttp

import (
	"net/http"
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
func RetryTransactor(retryCount int, shouldRetry ShouldRetryFunc, next func(*http.Request) (*http.Response, error)) func(*http.Request) (*http.Response, error) {
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
			if err == nil || !shouldRetry(err) {
				break
			}
		}

		return response, err
	}
}
