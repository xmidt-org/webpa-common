package xhttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTimeoutNoTimeout(t *testing.T, timeout time.Duration) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedResponse = httptest.NewRecorder()
		expectedRequest  = httptest.NewRequest("GET", "/", nil)
		nextCalled       = false
		next             = http.HandlerFunc(func(actualResponse http.ResponseWriter, actualRequest *http.Request) {
			nextCalled = true
			assert.Equal(expectedResponse, actualResponse)
			assert.Equal(expectedRequest, actualRequest)
		})

		constructor = Timeout(timeout)
	)

	require.NotNil(constructor)
	decorated := constructor(next)
	require.NotNil(decorated)

	decorated.ServeHTTP(expectedResponse, expectedRequest)
	assert.True(nextCalled)
}

func testTimeout(t *testing.T, timeout time.Duration) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedResponse = httptest.NewRecorder()
		originalRequest  = httptest.NewRequest("GET", "/", nil)
		capturedContext  context.Context
		next             = http.HandlerFunc(func(actualResponse http.ResponseWriter, actualRequest *http.Request) {
			assert.Equal(expectedResponse, actualResponse)
			assert.NotEqual(originalRequest, actualRequest)

			capturedContext = actualRequest.Context()
			assert.NotNil(capturedContext.Done())
			assert.Nil(capturedContext.Err())
			deadline, ok := capturedContext.Deadline()
			assert.False(deadline.IsZero())
			assert.True(ok)
		})

		constructor = Timeout(timeout)
	)

	require.NotNil(constructor)
	decorated := constructor(next)
	require.NotNil(decorated)

	decorated.ServeHTTP(expectedResponse, originalRequest)
	require.NotNil(capturedContext)

	assert.NotNil(capturedContext.Err())
	select {
	case <-capturedContext.Done():
		// passing
	default:
		assert.Fail("The decorator must cancel the context")
	}
}

func TestTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -1} {
		t.Run(fmt.Sprintf("timeout=%s", timeout), func(t *testing.T) { testTimeoutNoTimeout(t, timeout) })
	}

	for _, timeout := range []time.Duration{30 * time.Second, 15 * time.Hour} {
		t.Run(fmt.Sprintf("timeout=%s", timeout), func(t *testing.T) { testTimeout(t, timeout) })
	}
}
