// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xtimeout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
)

func testTimeoutHandlerPanic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		panicDelegate = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("expected")
		})

		handler = NewConstructor(Options{Timeout: time.Minute})(panicDelegate)

		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	require.NotNil(handler)

	defer func() {
		assert.Equal("expected", recover())
	}()

	handler.ServeHTTP(response, request)
	assert.Fail("ServeHTTP should have panicked")
}

func testTimeoutHandlerSuccess(t *testing.T) {
	const body = "success!"

	var (
		assert = assert.New(t)

		delegate = xhttp.Constant{
			Code: 299,
			// nolint: typecheck
			Header: http.Header{
				"X-Custom": {"value"},
				"X-Multi":  {"1", "2"},
			},
			Body: []byte(body),
		}

		handler = NewConstructor(Options{Timeout: time.Minute})(delegate)

		request  = httptest.NewRequest("GET", "/", nil)
		response = httptest.NewRecorder()
	)

	handler.ServeHTTP(response, request)
	assert.Equal(299, response.Code)
	assert.Equal([]string{"value"}, response.Header()["X-Custom"])
	assert.Equal([]string{"1", "2"}, response.Header()["X-Multi"])
	assert.Equal(body, response.Body.String())
}

func testTimeoutHandlerTimeout(t *testing.T) {
	const body = "timeout!"

	var (
		assert = assert.New(t)

		delegateReady = make(chan struct{})
		delegateBlock = make(chan struct{})
		delegate      = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			response.WriteHeader(599)
			response.Write([]byte("the delegate's write should have been discarded"))
			close(delegateReady)
			<-delegateBlock
		})

		timedOut = xhttp.Constant{
			Code: 499,
			// nolint: typecheck
			Header: http.Header{
				"X-Custom": {"value"},
				"X-Multi":  {"1", "2"},
			},
			Body: []byte(body),
		}

		handlerDone = make(chan struct{})
		handler     = NewConstructor(Options{Timeout: time.Hour, TimedOut: timedOut})(delegate)

		parentCtx, parentCancel = context.WithCancel(context.Background())
		request                 = httptest.NewRequest("GET", "/", nil).WithContext(parentCtx)
		response                = httptest.NewRecorder()
	)

	go func() {
		defer close(handlerDone)
		handler.ServeHTTP(response, request)
	}()

	select {
	case <-delegateReady:
		// passing
	case <-time.After(10 * time.Second):
		assert.Fail("Delegate was not called")
	}

	parentCancel() // simulate a timeout

	select {
	case <-handlerDone:
		// passing
	case <-time.After(10 * time.Second):
		assert.Fail("Handler did not complete")
	}

	assert.Equal(499, response.Code)
	assert.Equal([]string{"value"}, response.Header()["X-Custom"])
	assert.Equal([]string{"1", "2"}, response.Header()["X-Multi"])
	assert.Equal(body, response.Body.String())
}

func TestTimeoutHandler(t *testing.T) {
	t.Run("Panic", testTimeoutHandlerPanic)
	t.Run("Success", testTimeoutHandlerSuccess)
	t.Run("Timeout", testTimeoutHandlerTimeout)
}

func testNewConstructorNoTimeout(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		delegate    = xhttp.Constant{Code: 388}
		constructor = NewConstructor(Options{})
	)

	require.NotNil(constructor)
	assert.Equal(delegate, constructor(delegate))
}

func testNewConstructorDefaultTimedOut(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		delegate = xhttp.Constant{Code: 123}
		handler  = NewConstructor(Options{Timeout: time.Hour})(delegate)
	)

	th, ok := handler.(*timeoutHandler)
	require.True(ok)

	assert.Equal(time.Hour, th.timeout)
	assert.Equal(defaultTimedOut, th.timedOut)
	assert.Equal(delegate, th.next)
}

func TestNewConstructor(t *testing.T) {
	t.Run("NoTimeout", testNewConstructorNoTimeout)
	t.Run("DefaultTimedOut", testNewConstructorDefaultTimedOut)
}
