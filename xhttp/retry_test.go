package xhttp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func testRetryTransactorAllRetriesFail(t *testing.T, expectedInterval, configuredInterval time.Duration, retryCount int) {
	var (
		assert          = assert.New(t)
		require         = require.New(t)
		expectedRequest = httptest.NewRequest("GET", "/", nil)
		expectedError   = &net.DNSError{IsTemporary: true}
		counter         = generic.NewCounter("test")

		transactorCount = 0
		transactor      = func(actualRequest *http.Request) (*http.Response, error) {
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
}
