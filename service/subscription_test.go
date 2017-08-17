package service

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testSubscriptionWatchError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")
		registrar     = new(mockRegistrar)

		subscription = Subscription{
			Registrar: registrar,
			Listener: func([]string) {
				assert.Fail("The listener should not have been called")
			},
		}
	)

	registrar.On("Watch").Once().Return(nil, expectedError)

	assert.Equal(expectedError, subscription.Run())
	assert.Equal(ErrorNotRunning, subscription.Cancel())

	registrar.AssertExpectations(t)
}

func testSubscriptionListenerPanic(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")

		watch     = new(mockWatch)
		registrar = new(mockRegistrar)

		expectedEndpoints = []string{"expected endpoint"}
		listenerCalled    = make(chan struct{})

		subscription = Subscription{
			Registrar: registrar,
			Listener: func(endpoints []string) {
				defer close(listenerCalled)
				assert.Equal(expectedEndpoints, endpoints)
				panic(expectedError)
			},
		}
	)

	registrar.On("Watch").Once().Return(watch, nil)
	watch.On("Endpoints").Once().Return(expectedEndpoints)
	watch.On("Close")

	assert.NoError(subscription.Run())

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-listenerCalled:
		// Cancel should be idempotent
		subscription.Cancel()
		assert.Equal(ErrorNotRunning, subscription.Cancel())

		// passing case
		registrar.AssertExpectations(t)
		watch.AssertExpectations(t)

	case <-timer.C:
		assert.Fail("The listener was not called")
	}
}

func testSubscriptionNoTimeout(t *testing.T) {
	var (
		assert = assert.New(t)

		watch     = new(mockWatch)
		registrar = new(mockRegistrar)
		event     = make(chan struct{})

		expectedEndpoints = [][]string{
			[]string{"testSubscriptionNoTimeout1"},
			[]string{"testSubscriptionNoTimeout2", "testSubscriptionNoTimeout3"},
			[]string{"testSubscriptionNoTimeout4", "testSubscriptionNoTimeout5", "testSubscriptionNoTimeout6"},
		}

		actualCount  uint32
		listenerDone = make(chan struct{})

		subscription = Subscription{
			Registrar: registrar,
			Listener: func(endpoints []string) {
				assert.Equal(expectedEndpoints[actualCount], endpoints)
				newCount := atomic.AddUint32(&actualCount, 1)

				if int(newCount) == len(expectedEndpoints) {
					close(listenerDone)
				}
			},
		}
	)

	registrar.On("Watch").Once().Return(watch, nil)
	watch.On("Event").Return((<-chan struct{})(event))
	watch.On("IsClosed").Return(false)
	watch.On("Close")

	for _, endpoints := range expectedEndpoints {
		watch.On("Endpoints").Once().Return(endpoints)
	}

	assert.NoError(subscription.Run())
	assert.Equal(ErrorAlreadyRunning, subscription.Run())

	// do this one less than the expected endpoints, since the first element is the initial endpoints
	for repeat := 1; repeat < len(expectedEndpoints); repeat++ {
		event <- struct{}{}
	}

	timer := time.NewTimer(time.Second)
	select {
	case <-listenerDone:
		assert.Equal(len(expectedEndpoints), int(atomic.LoadUint32(&actualCount)))
		assert.NoError(subscription.Cancel())
		assert.Equal(ErrorNotRunning, subscription.Cancel())

		registrar.AssertExpectations(t)
		watch.AssertExpectations(t)

	case <-timer.C:
		assert.Fail("The listener did not receive all endpoints")
	}
}

func testSubscriptionWithTimeout(t *testing.T) {
	var (
		assert = assert.New(t)

		watch     = new(mockWatch)
		registrar = new(mockRegistrar)
		event     = make(chan struct{})

		expectedInitialEndpoints = []string{"initial"}

		// these are batches of endpoints.  the listener should receive only the last item in each batch.
		expectedEndpoints = [][][]string{
			[][]string{
				[]string{"batch1"},
				[]string{"batch1", "batch1"},
				[]string{"batch1", "batch1", "batch1"},
			},
			[][]string{
				[]string{"batch2"},
				[]string{"batch2", "batch2"},
				[]string{"batch2", "batch2", "batch2"},
			},
		}

		actualCount  uint32
		listenerDone = make(chan struct{})
		delay        = make(chan time.Time)

		subscription = Subscription{
			Registrar: registrar,
			After:     func(time.Duration) <-chan time.Time { return delay },
			Timeout:   time.Second,
			Listener: func(endpoints []string) {
				newCount := atomic.AddUint32(&actualCount, 1)
				if newCount == 1 {
					assert.Equal(expectedInitialEndpoints, endpoints)
				} else {
					batch := expectedEndpoints[newCount-2]
					assert.Equal(batch[len(batch)-1], endpoints)
				}

				if int(newCount) == len(expectedEndpoints)+1 {
					close(listenerDone)
				}
			},
		}
	)

	registrar.On("Watch").Once().Return(watch, nil)
	watch.On("Event").Return((<-chan struct{})(event))
	watch.On("IsClosed").Return(false)
	watch.On("Close")
	watch.On("Endpoints").Once().Return(expectedInitialEndpoints)

	for _, batch := range expectedEndpoints {
		for _, endpoints := range batch {
			watch.On("Endpoints").Once().Return(endpoints)
		}
	}

	assert.NoError(subscription.Run())
	assert.Equal(ErrorAlreadyRunning, subscription.Run())

	// for each batch, dispatch an event for each set of endpoints followed by a delay event
	for i, batch := range expectedEndpoints {
		for repeat := 0; repeat < len(batch); repeat++ {
			event <- struct{}{}
		}

		// don't send a time event after the last batch
		if i <= len(expectedEndpoints) {
			delay <- time.Now()
		}
	}

	timer := time.NewTimer(time.Second)
	select {
	case <-listenerDone:
		assert.Equal(len(expectedEndpoints)+1, int(atomic.LoadUint32(&actualCount)))
		assert.NoError(subscription.Cancel())
		assert.Equal(ErrorNotRunning, subscription.Cancel())

		registrar.AssertExpectations(t)
		watch.AssertExpectations(t)

	case <-timer.C:
		assert.Fail("The listener did not receive all endpoints")
	}
}

func TestSubscription(t *testing.T) {
	t.Run("WatchError", testSubscriptionWatchError)
	t.Run("ListenerPanic", testSubscriptionListenerPanic)
	t.Run("NoTimeout", testSubscriptionNoTimeout)
	t.Run("WithTimeout", testSubscriptionWithTimeout)
}
