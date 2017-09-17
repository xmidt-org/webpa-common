package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConcurrentNoCancellation(t *testing.T, concurrency int) {
	var (
		require          = require.New(t)
		assert           = assert.New(t)
		expectedRequest  = "expected request"
		expectedResponse = "expected response"

		nextCalled = false
		next       = func(ctx context.Context, value interface{}) (interface{}, error) {
			nextCalled = true
			assert.Equal(expectedRequest, value)
			return expectedResponse, nil
		}

		concurrent = Concurrent(concurrency, nil)
	)

	require.NotNil(concurrent)
	actualResponse, err := concurrent(next)(context.Background(), expectedRequest)
	assert.Equal(expectedResponse, actualResponse)
	assert.NoError(err)
}

func testConcurrentCancel(t *testing.T, concurrency int, timeoutError error) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithCancel(context.Background())
		expectedResponse    = "expected response"

		nextWaiting = new(sync.WaitGroup)
		nextBarrier = make(chan struct{})
		next        = func(ctx context.Context, value interface{}) (interface{}, error) {
			wait, ok := value.(func())
			if ok {
				wait()
			}

			return expectedResponse, nil
		}

		concurrent = Concurrent(concurrency, timeoutError)
	)

	require.NotNil(concurrent)
	endpoint := concurrent(next)

	// spawn enough goroutines to exhaust the semaphore
	nextWaiting.Add(concurrency)
	for r := 0; r < concurrency; r++ {
		go endpoint(expectedCtx, func() {
			nextWaiting.Done()
			<-nextBarrier
		})
	}

	// wait until we know the semaphore is exhausted, then cancel
	nextWaiting.Wait()
	cancel()

	// because the context is cancelled, subsequent calls should complete immediately
	actualResponse, err := endpoint(expectedCtx, "request")
	assert.Nil(actualResponse)
	assert.NotNil(err)

	if timeoutError != nil {
		assert.Equal(timeoutError, err)
	} else {
		assert.Equal(context.Canceled, err)
	}

	close(nextBarrier)
}

func TestConcurrent(t *testing.T) {
	t.Run("NoCancellation", func(t *testing.T) {
		for _, c := range []int{1, 10, 15, 100} {
			t.Run(fmt.Sprintf("Concurrency=%d", c), func(t *testing.T) {
				testConcurrentNoCancellation(t, c)
			})
		}
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Run("NilTimeoutError", func(t *testing.T) {
			for _, c := range []int{1, 10, 15, 100} {
				t.Run(fmt.Sprintf("Concurrency=%d", c), func(t *testing.T) {
					testConcurrentCancel(t, c, nil)
				})
			}
		})

		t.Run("WithTimeoutError", func(t *testing.T) {
			for _, c := range []int{1, 10, 15, 100} {
				t.Run(fmt.Sprintf("Concurrency=%d", c), func(t *testing.T) {
					testConcurrentCancel(t, c, errors.New("expected timeout error"))
				})
			}
		})
	})
}
