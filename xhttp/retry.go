/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package xhttp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

const DefaultRetryInterval = time.Second

// temporaryError is the expected interface for a (possibly) temporary error.
// Several of the error types in the net package implicitely implement this interface,
// for example net.DNSError.
type temporaryError interface {
	Temporary() bool
}

// ShouldRetryFunc is a predicate for determining if the error returned from an HTTP transaction
// should be retried.
type ShouldRetryFunc func(error) bool

// ShouldRetryStatusFunc is a predicate for determining if the status coded returned from an HTTP transaction
// should be retried.
type ShouldRetryStatusFunc func(int) bool

// DefaultShouldRetry is the default retry predicate.  It returns true if and only if err exposes a Temporary() bool
// method and that method returns true.  That means, for example, that for a net.DNSError with the temporary flag set to true
// this predicate also returns true.
func DefaultShouldRetry(err error) bool {
	if temp, ok := err.(temporaryError); ok {
		return temp.Temporary()
	}

	return false
}

// DefaultShouldRetryStatus is the default retry predicate. It returns false on all status codes
// aka. it will never retry
func DefaultShouldRetryStatus(status int) bool {
	return false
}

// RetryOptions are the configuration options for a retry transactor
type RetryOptions struct {
	// Logger is the go-kit logger to use.  Defaults to logging.DefaultLogger() if unset.
	Logger log.Logger

	// Retries is the count of retries.  If not positive, then no transactor decoration is performed.
	Retries int

	// Interval is the time between retries.  If not set, DefaultRetryInterval is used.
	Interval time.Duration

	// Sleep is function used to wait out a duration.  If unset, time.Sleep is used.
	Sleep func(time.Duration)

	// ShouldRetry is the retry predicate.  Defaults to DefaultShouldRetry if unset.
	ShouldRetry ShouldRetryFunc

	// ShouldRetryStatus is the retry predicate.  Defaults to DefaultShouldRetry if unset.
	ShouldRetryStatus ShouldRetryStatusFunc

	// Counter is the counter for total retries.  If unset, no metrics are collected on retries.
	Counter metrics.Counter

	// UpdateRequest provides the ability to update the request before it is sent. default is noop
	UpdateRequest func(*http.Request)
}

// RetryTransactor returns an HTTP transactor function, of the same signature as http.Client.Do, that
// retries a certain number of times.  Note that net/http.RoundTripper.RoundTrip also is of this signature,
// so this decorator can be used with a RoundTripper or an http.Client equally well.
//
// If o.Retries is nonpositive, next is returned undecorated.
func RetryTransactor(o RetryOptions, next func(*http.Request) (*http.Response, error)) func(*http.Request) (*http.Response, error) {
	if o.Retries < 1 {
		return next
	}

	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	if o.Counter == nil {
		o.Counter = discard.NewCounter()
	}

	if o.ShouldRetry == nil {
		o.ShouldRetry = DefaultShouldRetry
	}

	if o.ShouldRetryStatus == nil {
		o.ShouldRetryStatus = DefaultShouldRetryStatus
	}

	if o.UpdateRequest == nil {
		//noop
		o.UpdateRequest = func(*http.Request) {}
	}

	if o.Interval < 1 {
		o.Interval = DefaultRetryInterval
	}

	if o.Sleep == nil {
		o.Sleep = time.Sleep
	}

	return func(request *http.Request) (*http.Response, error) {
		if err := EnsureRewindable(request); err != nil {
			return nil, err
		}
		var statusCode int

		// initial attempt:
		response, err := next(request)
		if response != nil {
			statusCode = response.StatusCode
		}

		for r := 0; r < o.Retries && ((err != nil && o.ShouldRetry(err)) || o.ShouldRetryStatus(statusCode)); r++ {
			o.Counter.Add(1.0)
			o.Sleep(o.Interval)
			o.Logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "retrying HTTP transaction", "url", request.URL.String(), logging.ErrorKey(), err, "retry", r+1, "statusCode", statusCode)

			if err := Rewind(request); err != nil {
				return nil, err
			}

			o.UpdateRequest(request)
			response, err = next(request)
			if response != nil {
				statusCode = response.StatusCode
			}
		}

		if err != nil {
			o.Logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "All HTTP transaction retries failed", "url", request.URL.String(), logging.ErrorKey(), err, "retries", o.Retries)
		}

		return response, err
	}
}

func IsTemporary(err error) bool {
	type temporary interface {
		Temporary() bool
	}
	var te temporary
	if errors.As(err, &te) {
		return te.Temporary()
	}
	return false
}

func ShouldRetry(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return IsTemporary(err)
}

func RetryCodes(i int) bool {
	switch i {
	case http.StatusRequestTimeout:
		return true
	case http.StatusTooManyRequests:
		return true
	case http.StatusGatewayTimeout:
		return true
	}
	return false
}
