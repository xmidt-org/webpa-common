package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBusyBadMaxClients(t *testing.T, maxClients int64) {
	assert := assert.New(t)

	assert.Panics(func() {
		Busy(maxClients, nil)
	})

	assert.Panics(func() {
		Busy(maxClients, errors.New("custom busy error"))
	})
}

func testBusyClientCounter(t *testing.T, maxClients int64, busyError error) {
	var (
		assert      = assert.New(t)
		expectedCtx = context.WithValue(context.Background(), "foo", "bar")

		endpointGate     = make(chan struct{})
		endpointsWaiting = new(sync.WaitGroup)
		endpointsExiting = make(chan struct{}, maxClients)

		busyEndpoint = Busy(maxClients, busyError)(func(ctx context.Context, value interface{}) (interface{}, error) {
			assert.Equal(expectedCtx, ctx)
			if value == "blocking" {
				endpointsWaiting.Done()
				<-endpointGate
			}

			return "done", nil
		})
	)

	// exhaust the Busy max number of clients
	endpointsWaiting.Add(int(maxClients))
	for r := int64(0); r < maxClients; r++ {
		go func() {
			defer func() {
				endpointsExiting <- struct{}{}
			}()

			actual, err := busyEndpoint(expectedCtx, "blocking")
			assert.Equal("done", actual)
			assert.NoError(err)
		}()
	}

	// while we have a known number of clients blocking, attempt to make another call
	endpointsWaiting.Wait()
	actual, err := busyEndpoint(expectedCtx, "rejected")
	assert.Nil(actual)
	assert.Error(err)

	if busyError != nil {
		assert.Equal(busyError, err)
	}

	// now wait until any blocked endpoint is done, and try to execute an endpoint
	close(endpointGate)
	<-endpointsExiting
	actual, err = busyEndpoint(expectedCtx, "succeed")
	assert.Equal("done", actual)
	assert.NoError(err)
}

func TestBusy(t *testing.T) {
	t.Run("BadMaxClients", func(t *testing.T) {
		testBusyBadMaxClients(t, 0)
		testBusyBadMaxClients(t, -1)
	})

	t.Run("ClientCounter", func(t *testing.T) {
		t.Run("NilBusyError", func(t *testing.T) {
			for _, c := range []int64{1, 10, 100} {
				t.Run(fmt.Sprintf("MaxClients=%d", c), func(t *testing.T) {
					testBusyClientCounter(t, c, nil)
				})
			}
		})

		t.Run("CustomBusyError", func(t *testing.T) {
			for _, c := range []int64{1, 10, 100} {
				t.Run(fmt.Sprintf("MaxClients=%d", c), func(t *testing.T) {
					testBusyClientCounter(t, c, errors.New("custom busy error"))
				})
			}
		})
	})
}
