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
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

func testShouldRetry(t *testing.T, shouldRetry ShouldRetryFunc, candidate error, expected bool) {
	assert := assert.New(t)
	assert.Equal(expected, shouldRetry(candidate))
}

func TestDefaultShouldRetry(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		testShouldRetry(t, DefaultShouldRetry, nil, false)
	})

	t.Run("DNSError", func(t *testing.T) {
		testShouldRetry(t, DefaultShouldRetry, &net.DNSError{IsTemporary: false}, false)
		testShouldRetry(t, DefaultShouldRetry, &net.DNSError{IsTemporary: true}, true)
	})
}

func testRetryTransactorDefaultLogger(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		transactor = func(*http.Request) (*http.Response, error) {
			transactorCalled = true
			return nil, nil
		}

		retry = RetryTransactor(RetryOptions{Retries: 1}, transactor)
	)

	require.NotNil(retry)
	retry(httptest.NewRequest("GET", "/", nil))
	assert.True(transactorCalled)
}

func testRetryTransactorNoRetries(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		transactor = func(*http.Request) (*http.Response, error) {
			transactorCalled = true
			return nil, nil
		}

		retry = RetryTransactor(RetryOptions{}, transactor)
	)

	require.NotNil(retry)
	retry(httptest.NewRequest("GET", "/", nil))
	assert.True(transactorCalled)
}

func testRetryTransactorStatus(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		transactorCount = 0
		statusCheck     = 0
		transactor      = func(*http.Request) (*http.Response, error) {
			response := http.Response{
				StatusCode: 429 + transactorCount,
			}
			transactorCount++
			return &response, nil
		}

		retry = RetryTransactor(RetryOptions{
			Retries: 5,
			ShouldRetryStatus: func(status int) bool {
				statusCheck++
				return status == 429
			},
		}, transactor)
	)

	require.NotNil(retry)
	retry(httptest.NewRequest("GET", "/", nil))
	assert.Equal(2, transactorCount)
	assert.Equal(2, statusCheck)
}

func testRetryTransactorAllRetriesFail(t *testing.T, expectedInterval, configuredInterval time.Duration, retryCount int) {
	var (
		assert          = assert.New(t)
		require         = require.New(t)
		expectedRequest = httptest.NewRequest("GET", "/", nil)
		expectedError   = &net.DNSError{IsTemporary: true}
		counter         = generic.NewCounter("test")
		urls            = map[string]int{}

		transactorCount = 0
		transactor      = func(actualRequest *http.Request) (*http.Response, error) {
			if _, ok := urls[actualRequest.URL.Path]; ok {
				urls[actualRequest.URL.Path]++
			} else {
				urls[actualRequest.URL.Path] = 1
			}
			transactorCount++
			assert.True(expectedRequest == actualRequest)
			return nil, expectedError
		}

		slept = 0
		retry = RetryTransactor(
			RetryOptions{
				Logger:   logging.NewTestLogger(nil, t),
				Retries:  retryCount,
				Counter:  counter,
				Interval: configuredInterval,
				Sleep: func(actualInterval time.Duration) {
					slept++
					assert.Equal(expectedInterval, actualInterval)
				},
				UpdateRequest: func(request *http.Request) {
					if _, ok := urls[request.URL.Path]; ok {
						request.URL.Path += "a"
					}
				},
			},
			transactor,
		)
	)

	require.NotNil(retry)
	actualResponse, actualError := retry(expectedRequest)
	assert.Nil(actualResponse)
	assert.Equal(expectedError, actualError)
	assert.Equal(1+retryCount, transactorCount)
	assert.Equal(float64(retryCount), counter.Value())
	assert.Equal(retryCount, slept)
	for _, v := range urls {
		assert.Equal(1, v)
	}
}

func testRetryTransactorFirstSucceeds(t *testing.T, retryCount int) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		expectedRequest  = httptest.NewRequest("GET", "/", nil)
		expectedResponse = new(http.Response)
		counter          = generic.NewCounter("test")

		transactorCount = 0
		transactor      = func(actualRequest *http.Request) (*http.Response, error) {
			transactorCount++
			assert.True(expectedRequest == actualRequest)
			return expectedResponse, nil
		}

		retry = RetryTransactor(
			RetryOptions{
				Logger:  logging.NewTestLogger(nil, t),
				Retries: retryCount,
				Counter: counter,
				Sleep: func(d time.Duration) {
					assert.Fail("Sleep should not have been called")
				},
			},
			transactor,
		)
	)

	require.NotNil(retry)
	actualResponse, actualError := retry(expectedRequest)
	assert.True(expectedResponse == actualResponse)
	assert.NoError(actualError)
	assert.Equal(1, transactorCount)
	assert.Zero(counter.Value())
}

func testRetryTransactorNotRewindable(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		body          = new(mockReader)
		expectedError = errors.New("expected")

		retry = RetryTransactor(
			RetryOptions{
				Logger:  logging.NewTestLogger(nil, t),
				Retries: 2,
			},
			func(*http.Request) (*http.Response, error) {
				assert.Fail("The decorated transactor should not have been called")
				return nil, nil
			},
		)
	)

	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	require.NotNil(retry)
	response, actualError := retry(&http.Request{Body: ioutil.NopCloser(body)})
	assert.Nil(response)
	assert.Equal(expectedError, actualError)

	body.AssertExpectations(t)
}

func testRetryTransactorRewindError(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedError = errors.New("expected")

		retry = RetryTransactor(
			RetryOptions{
				Logger:  logging.NewTestLogger(nil, t),
				Retries: 2,
				Sleep:   func(time.Duration) {},
			},
			func(*http.Request) (*http.Response, error) {
				return nil, &net.DNSError{IsTemporary: true}
			},
		)

		r = httptest.NewRequest("POST", "/", nil)
	)

	r.GetBody = func() (io.ReadCloser, error) {
		return nil, expectedError
	}

	require.NotNil(retry)
	response, actualError := retry(r)
	assert.Nil(response)
	assert.Equal(expectedError, actualError)
}

func TestRetryTransactor(t *testing.T) {
	t.Run("DefaultLogger", testRetryTransactorDefaultLogger)
	t.Run("NoRetries", testRetryTransactorNoRetries)

	t.Run("AllRetriesFail", func(t *testing.T) {
		for _, retryCount := range []int{1, 2, 5} {
			t.Run(fmt.Sprintf("RetryCount=%d", retryCount), func(t *testing.T) {
				testRetryTransactorAllRetriesFail(t, time.Second, 0, retryCount)
				testRetryTransactorAllRetriesFail(t, 10*time.Minute, 10*time.Minute, retryCount)
			})
		}
	})

	t.Run("FirstSucceeds", func(t *testing.T) {
		for _, retryCount := range []int{1, 2, 5} {
			t.Run(fmt.Sprintf("RetryCount=%d", retryCount), func(t *testing.T) { testRetryTransactorFirstSucceeds(t, retryCount) })
		}
	})

	t.Run("NotRewindable", testRetryTransactorNotRewindable)
	t.Run("RewindError", testRetryTransactorRewindError)
	t.Run("StatusRetry", testRetryTransactorStatus)
}

func TestRetryCodes(t *testing.T) {
	tcs := []struct {
		desc           string
		httpCode       int
		expectedResult bool
	}{
		{
			desc:           "StatusRequestTimeout",
			httpCode:       408,
			expectedResult: true,
		},
		{
			desc:           "StatusTooManyRequests",
			httpCode:       429,
			expectedResult: true,
		},
		{
			desc:           "StatusGatewayTimeout",
			httpCode:       504,
			expectedResult: true,
		},
		{
			desc:           "Random Code Failure",
			httpCode:       400,
			expectedResult: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			res := RetryCodes(tc.httpCode)
			assert.Equal(res, tc.expectedResult)
		})
	}
}

func TestShouldRetry(t *testing.T) {
	var mockTempErr mockTempError
	tcs := []struct {
		desc           string
		errCase        error
		expectedResult bool
	}{
		{
			desc:           "DeadlineExceeded Case",
			errCase:        context.DeadlineExceeded,
			expectedResult: false,
		},
		{
			desc:           "False Temporary Error Case",
			errCase:        errors.New("not temp"),
			expectedResult: false,
		},
		{
			desc:           "True Temporary Error Case",
			errCase:        mockTempErr,
			expectedResult: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			res := ShouldRetry(tc.errCase)
			assert.Equal(res, tc.expectedResult)
		})
	}
}
