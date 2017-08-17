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
	watch.On("Close").Once()

	assert.NoError(subscription.Run())

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-listenerCalled:
		// passing case
		registrar.AssertExpectations(t)
		watch.AssertExpectations(t)

		// Cancel should be idempotent
		subscription.Cancel()
		assert.Equal(ErrorNotRunning, subscription.Cancel())

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

	// now the rest of the endpoints
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

		watch     = NewTestWatch(t)
		registrar = new(mockRegistrar)
		delay     = make(chan time.Time)

		expectedEndpoints = [][]string{
			[]string{"testSubscriptionWithTimeout1"},
			[]string{"testSubscriptionWithTimeout2", "testSubscriptionWithTimeout3"},
			[]string{"testSubscriptionWithTimeout4", "testSubscriptionWithTimeout5", "testSubscriptionWithTimeout6"},
		}

		afterCalled     = make(chan struct{})
		expectedTimeout = 12856 * time.Second
		listenerOutput  = make(chan []string, 1)
		subscription    = Subscription{
			Registrar: registrar,
			Timeout:   expectedTimeout,
			After: func(timeout time.Duration) <-chan time.Time {
				t.Logf("After function called with %s", timeout)
				defer close(afterCalled)
				assert.Equal(expectedTimeout, timeout)
				return delay
			},
			Listener: func(endpoints []string) {
				listenerOutput <- endpoints
			},
		}
	)

	registrar.On("Watch").Once().Return(watch, nil)

	assert.NoError(subscription.Run())
	assert.Equal(ErrorAlreadyRunning, subscription.Run())

	// this simulates a succession of events
	for _, endpoints := range expectedEndpoints {
		t.Logf("next endpoints: %s", endpoints)
		watch.NextEndpoints(endpoints)

		select {
		case <-listenerOutput:
			assert.Fail("The listener should not have been invoked")
		default:
			// passing
		}
	}

	// ensure that our After function was called
	select {
	case <-afterCalled:
		// passing
	default:
		assert.Fail("The After function should have been called")
	}

	// simulate the timer elapsing
	delay <- time.Now()
	assert.Equal(expectedEndpoints[len(expectedEndpoints)-1], <-listenerOutput)

	assert.NoError(subscription.Cancel())
	assert.True(watch.IsClosed())
	assert.Equal(ErrorNotRunning, subscription.Cancel())
	assert.True(watch.IsClosed())

	registrar.AssertExpectations(t)
}

func TestSubscription(t *testing.T) {
	t.Run("WatchError", testSubscriptionWatchError)
	t.Run("ListenerPanic", testSubscriptionListenerPanic)
	t.Run("NoTimeout", testSubscriptionNoTimeout)
	/*
		t.Run("WithTimeout", testSubscriptionWithTimeout)
	*/
}
