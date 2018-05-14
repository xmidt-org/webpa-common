package xhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBusyInvalidMaxTransactions(t *testing.T) {
	assert := assert.New(t)

	assert.Panics(func() {
		Busy(0)
	})

	assert.Panics(func() {
		Busy(-1)
	})
}

func testBusySimple(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		next = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(231)
		})

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		decorated = Busy(1)(next)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(231, response.Code)
}

func testBusyCancelation(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		firstReceived = make(chan struct{})
		firstWaiting  = make(chan struct{})
		firstComplete = make(chan struct{})

		next = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			close(firstReceived)
			<-firstWaiting
			response.WriteHeader(299)
		})

		decorated = Busy(1)(next)
	)

	require.NotNil(decorated)

	// spawn a "long running" in flight HTTP transaction
	go func() {
		defer close(firstComplete)

		var (
			response = httptest.NewRecorder()
			request  = httptest.NewRequest("GET", "/longrunning", nil)
		)

		decorated.ServeHTTP(response, request)
		assert.Equal(299, response.Code)
	}()

	<-firstReceived

	// now any HTTP transaction should be held up waiting on the semaphore

	var (
		ctx, cancel = context.WithCancel(context.Background())
		response    = httptest.NewRecorder()
		request     = httptest.NewRequest("GET", "/rejected", nil).WithContext(ctx)
	)

	cancel()
	decorated.ServeHTTP(response, request)
	assert.Equal(http.StatusServiceUnavailable, response.Code)

	close(firstWaiting)
	<-firstComplete
}

func TestBusy(t *testing.T) {
	t.Run("InvalidMaxTransactions", testBusyInvalidMaxTransactions)
	t.Run("Simple", testBusySimple)
	t.Run("Cancelation", testBusyCancelation)
}
