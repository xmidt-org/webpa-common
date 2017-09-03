package service

import (
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testSubscribeNoDelay(t *testing.T) {
	var (
		assert    = assert.New(t)
		instancer = new(mockInstancer)

		registeredChannel chan<- sd.Event
		registerCalled    = make(chan struct{})
		deregisterCalled  = make(chan struct{})

		options = &Options{
			Logger: logging.NewTestLogger(&logging.Options{Level: "debug", JSON: true}, t),
			After: func(time.Duration) <-chan time.Time {
				assert.Fail("The after function should not have been called")
				return nil
			},
		}
	)

	instancer.On("Register", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		registeredChannel = ch
		return true
	})).Run(func(mock.Arguments) { close(registerCalled) }).Once()

	instancer.On("Deregister", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		assert.Equal(registeredChannel, ch)
		return true
	})).Run(func(mock.Arguments) { close(deregisterCalled) }).Once()

	// start the subscription under test
	sub := Subscribe(options, instancer)
	assert.NotEmpty(sub.(*subscription).String())
	assert.Zero(len(sub.Updates()))

	select {
	case <-registerCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Register was not called")
	}

	registeredChannel <- sd.Event{Err: errors.New("expected")}
	assert.Zero(len(sub.Updates()))

	registeredChannel <- sd.Event{Instances: []string{"localhost:8888"}}
	select {
	case accessor := <-sub.Updates():
		instance, err := accessor.Get([]byte("some key"))
		assert.Equal("localhost:8888", instance)
		assert.NoError(err)

	case <-sub.Stopped():
		assert.Fail("The subscription should not have stopped")

	case <-time.After(time.Second):
		assert.Fail("No accessor update occurred")
	}

	registeredChannel <- sd.Event{Instances: []string{"localhost:1234"}}
	select {
	case accessor := <-sub.Updates():
		instance, err := accessor.Get([]byte("some key"))
		assert.Equal("localhost:1234", instance)
		assert.NoError(err)

	case <-sub.Stopped():
		assert.Fail("The subscription should not have stopped")

	case <-time.After(time.Second):
		assert.Fail("No accessor update occurred")
	}

	sub.Stop()

	select {
	case <-deregisterCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Deregister was not called")
	}

	sub.Stop() // idempotency
	instancer.AssertExpectations(t)
}

func testSubscribeDelay(t *testing.T) {
	var (
		assert    = assert.New(t)
		instancer = new(mockInstancer)
		delay     = make(chan time.Time, 1)

		registeredChannel chan<- sd.Event
		registerCalled    = make(chan struct{})
		deregisterCalled  = make(chan struct{})

		options = &Options{
			Logger:      logging.NewTestLogger(&logging.Options{Level: "debug", JSON: true}, t),
			UpdateDelay: 5 * time.Minute,
			After: func(d time.Duration) <-chan time.Time {
				assert.Equal(5*time.Minute, d)
				return delay
			},
		}
	)

	instancer.On("Register", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		registeredChannel = ch
		return true
	})).Run(func(mock.Arguments) { close(registerCalled) }).Once()

	instancer.On("Deregister", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		assert.Equal(registeredChannel, ch)
		return true
	})).Run(func(mock.Arguments) { close(deregisterCalled) }).Once()

	// start the subscription under test
	sub := Subscribe(options, instancer)
	assert.NotEmpty(sub.(*subscription).String())
	assert.Zero(len(sub.Updates()))

	select {
	case <-registerCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Register was not called")
	}

	registeredChannel <- sd.Event{Err: errors.New("expected")}
	assert.Zero(len(sub.Updates()))

	// the very first event should be dispatched immediately
	registeredChannel <- sd.Event{Instances: []string{"localhost:8888"}}

	select {
	case accessor := <-sub.Updates():
		instance, err := accessor.Get([]byte("some key"))
		assert.Equal("localhost:8888", instance)
		assert.NoError(err)

	case <-sub.Stopped():
		assert.Fail("The subscription should not have stopped")

	case <-time.After(time.Second):
		assert.Fail("No accessor update occurred")
	}

	registeredChannel <- sd.Event{Instances: []string{"localhost:1234"}}

	select {
	case <-sub.Updates():
		assert.Fail("No updates should have been sent before the delay expired")

	case <-sub.Stopped():
		assert.Fail("The subscription should not have stopped")

	case <-time.After(250 * time.Millisecond):
		// passing
	}

	registeredChannel <- sd.Event{Instances: []string{"localhost:4321"}}

	delay <- time.Now()
	select {
	case accessor := <-sub.Updates():
		instance, err := accessor.Get([]byte("some key"))
		assert.Equal("localhost:4321", instance)
		assert.NoError(err)

	case <-sub.Stopped():
		assert.Fail("The subscription should not have stopped")

	case <-time.After(time.Second):
		assert.Fail("No accessor update occurred")
	}

	sub.Stop()

	select {
	case <-deregisterCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Deregister was not called")
	}

	sub.Stop() // idempotency
	instancer.AssertExpectations(t)
}

func testSubscribeMonitorPanic(t *testing.T) {
	var (
		assert    = assert.New(t)
		instancer = new(mockInstancer)

		registeredChannel chan<- sd.Event
		registerCalled    = make(chan struct{})
		deregisterCalled  = make(chan struct{})

		options = &Options{
			Logger: logging.NewTestLogger(&logging.Options{Level: "debug", JSON: true}, t),
			After: func(time.Duration) <-chan time.Time {
				assert.Fail("The after function should not have been called")
				return nil
			},
			InstancesFilter: func([]string) []string {
				panic("expected")
			},
		}
	)

	instancer.On("Register", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		registeredChannel = ch
		return true
	})).Run(func(mock.Arguments) { close(registerCalled) }).Once()

	instancer.On("Deregister", mock.MatchedBy(func(ch chan<- sd.Event) bool {
		assert.Equal(registeredChannel, ch)
		return true
	})).Run(func(mock.Arguments) { close(deregisterCalled) }).Once()

	// start the subscription under test
	sub := Subscribe(options, instancer)
	assert.NotEmpty(sub.(*subscription).String())
	assert.Zero(len(sub.Updates()))

	select {
	case <-registerCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Register was not called")
	}

	registeredChannel <- sd.Event{Err: errors.New("expected")}
	assert.Zero(len(sub.Updates()))

	// this should cause the panic
	registeredChannel <- sd.Event{Instances: []string{"localhost:8888"}}
	assert.Zero(len(sub.Updates()))
	select {
	case <-sub.Stopped():
		// passing

	case <-time.After(time.Second):
		assert.Fail("The subscription should have stopped itself after a panic")
	}

	select {
	case <-deregisterCalled:
		// passing
	case <-time.After(time.Second):
		assert.Fail("Instancer.Deregister was not called")
	}

	sub.Stop() // idempotency
	instancer.AssertExpectations(t)
}

func TestSubscribe(t *testing.T) {
	t.Run("NoDelay", testSubscribeNoDelay)
	t.Run("Delay", testSubscribeDelay)
	t.Run("MonitorPanic", testSubscribeMonitorPanic)
}
