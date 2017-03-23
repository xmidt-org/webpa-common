package service

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func testSubscriptionWatchError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("testSubscriptionWatchError")
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
		expectedError = errors.New("testSubscriptionListenerPanic")

		watch     = NewTestWatch(t)
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

	assert.NoError(subscription.Run())
	assert.Equal(ErrorAlreadyRunning, subscription.Run())

	// simulate one event, which should dispatch then panic
	watch.NextEndpoints(expectedEndpoints)
	<-listenerCalled

	// the monitor goroutine calls Cancel, so we can't guarantee that invoking
	// Cancel here will happen before (or after) that call.
	err := subscription.Cancel()
	assert.True(err == nil || err == ErrorNotRunning)
	assert.True(watch.IsClosed())

	// However, a second Cancel will always behave the same
	assert.Equal(ErrorNotRunning, subscription.Cancel())
	assert.True(watch.IsClosed())

	registrar.AssertExpectations(t)
}

func testSubscriptionNoTimeout(t *testing.T) {
	var (
		assert = assert.New(t)

		watch     = NewTestWatch(t)
		registrar = new(mockRegistrar)

		expectedEndpoints = [][]string{
			[]string{"testSubscriptionNoTimeout1"},
			[]string{"testSubscriptionNoTimeout2", "testSubscriptionNoTimeout3"},
			[]string{"testSubscriptionNoTimeout4", "testSubscriptionNoTimeout5", "testSubscriptionNoTimeout6"},
		}

		listenerOutput = make(chan []string, 1)
		subscription   = Subscription{
			Registrar: registrar,
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
		watch.NextEndpoints(endpoints)
		assert.Equal(endpoints, <-listenerOutput)
	}

	assert.NoError(subscription.Cancel())
	assert.True(watch.IsClosed())
	assert.Equal(ErrorNotRunning, subscription.Cancel())
	assert.True(watch.IsClosed())

	registrar.AssertExpectations(t)
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
	t.Run("WithTimeout", testSubscriptionWithTimeout)
}
