package xhttp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
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

func testRetryTransactorNoRetries(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		transactorCalled = false

		transactor = func(*http.Request) (*http.Response, error) {
			transactorCalled = true
			return nil, nil
		}

		retry = RetryTransactor(logging.NewTestLogger(nil, t), 0, nil, transactor)
	)

	require.NotNil(retry)
	retry(httptest.NewRequest("GET", "/", nil))
	assert.True(transactorCalled)
}

func testRetryTransactorAllRetriesFail(t *testing.T, retryCount int) {
	var (
		assert          = assert.New(t)
		require         = require.New(t)
		expectedRequest = httptest.NewRequest("GET", "/", nil)
		expectedError   = &net.DNSError{IsTemporary: true}

		transactorCount = 0
		transactor      = func(actualRequest *http.Request) (*http.Response, error) {
			transactorCount++
			assert.True(expectedRequest == actualRequest)
			return nil, expectedError
		}

		retry = RetryTransactor(logging.NewTestLogger(nil, t), retryCount, nil, transactor)
	)

	require.NotNil(retry)
	actualResponse, actualError := retry(expectedRequest)
	assert.Nil(actualResponse)
	assert.Equal(expectedError, actualError)
	assert.Equal(1+retryCount, transactorCount)
}

func testRetryTransactorFirstSucceeds(t *testing.T, retryCount int) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		expectedRequest  = httptest.NewRequest("GET", "/", nil)
		expectedResponse = new(http.Response)

		transactorCount = 0
		transactor      = func(actualRequest *http.Request) (*http.Response, error) {
			transactorCount++
			assert.True(expectedRequest == actualRequest)
			return expectedResponse, nil
		}

		retry = RetryTransactor(logging.NewTestLogger(nil, t), retryCount, nil, transactor)
	)

	require.NotNil(retry)
	actualResponse, actualError := retry(expectedRequest)
	assert.True(expectedResponse == actualResponse)
	assert.NoError(actualError)
	assert.Equal(1, transactorCount)
}

func TestRetryTransactor(t *testing.T) {
	t.Run("NoRetries", testRetryTransactorNoRetries)

	t.Run("AllRetriesFail", func(t *testing.T) {
		for _, retryCount := range []int{1, 2, 5} {
			t.Run(fmt.Sprintf("RetryCount=%d", retryCount), func(t *testing.T) { testRetryTransactorAllRetriesFail(t, retryCount) })
		}
	})

	t.Run("FirstSucceeds", func(t *testing.T) {
		for _, retryCount := range []int{1, 2, 5} {
			t.Run(fmt.Sprintf("RetryCount=%d", retryCount), func(t *testing.T) { testRetryTransactorFirstSucceeds(t, retryCount) })
		}
	})
}
