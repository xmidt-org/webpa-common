package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/servicemock"
	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testNewNoInstances(t *testing.T) {
	var (
		assert = assert.New(t)

		m, err = New(
			WithLogger(nil),
			WithFilter(NopFilter),
			WithClosed(nil),
			WithListeners(),
		)
	)

	assert.Nil(m)
	assert.Error(err)
}

func testNewStop(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		instancer         = new(servicemock.Instancer)
		listener          = new(mockListener)
		registerQueue     = make(chan chan<- sd.Event, 1)
		sdEvents          chan<- sd.Event
		expectedError     = errors.New("expected")
		expectedInstances = []string{"instance1", "instance2"}

		monitorEvents = make(chan Event, 5)
	)

	instancer.On("Register", mock.AnythingOfType("chan<- sd.Event")).
		Run(func(arguments mock.Arguments) {
			registerQueue <- arguments.Get(0).(chan<- sd.Event)
		}).Once()

	instancer.On("Deregister", mock.AnythingOfType("chan<- sd.Event")).
		Run(func(arguments mock.Arguments) {
			registerQueue <- arguments.Get(0).(chan<- sd.Event)
		}).Once()

	listener.On("MonitorEvent", mock.MatchedBy(func(Event) bool { return true })).Run(func(arguments mock.Arguments) {
		monitorEvents <- arguments.Get(0).(Event)
	})

	m, err := New(
		WithLogger(logger),
		WithFilter(nil),
		WithListeners(listener),
		WithInstancers(service.Instancers{"test": instancer}),
	)

	require.NoError(err)
	require.NotNil(m)

	select {
	case sdEvents = <-registerQueue:
	case <-time.After(5 * time.Second):
		m.Stop()
		require.Fail("Failed to receive registered event channel")
		return
	}

	sdEvents <- sd.Event{Err: expectedError}
	select {
	case event := <-monitorEvents:
		assert.Equal("test", event.Key)
		assert.Equal(expectedError, event.Err)
		assert.Len(event.Instances, 0)
		assert.False(event.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("Failed to receive monitor event")
	}

	sdEvents <- sd.Event{Instances: expectedInstances}
	select {
	case event := <-monitorEvents:
		assert.Equal("test", event.Key)
		assert.NoError(event.Err)
		assert.Equal(expectedInstances, event.Instances)
		assert.False(event.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("Failed to receive monitor event")
	}

	m.Stop()
	select {
	case deregistered := <-registerQueue:
		assert.Equal(sdEvents, deregistered)
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to deregister")
	}

	select {
	case finalEvent := <-monitorEvents:
		assert.Equal("test", finalEvent.Key)
		assert.NoError(finalEvent.Err)
		assert.Len(finalEvent.Instances, 0)
		assert.True(finalEvent.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("No stopped event received")
	}

	select {
	case <-m.Stopped():
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to signal stopped channel")
	}

	// idempotency
	m.Stop()
	select {
	case <-m.Stopped():
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to signal stopped channel")
	}

	instancer.AssertExpectations(t)
	listener.AssertExpectations(t)
}

func testNewWithEnvironment(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		instancer         = new(servicemock.Instancer)
		listener          = new(mockListener)
		registerQueue     = make(chan chan<- sd.Event, 1)
		sdEvents          chan<- sd.Event
		expectedError     = errors.New("expected")
		expectedInstances = []string{"instance1", "instance2"}

		monitorEvents = make(chan Event, 5)
	)

	e := service.NewEnvironment(
		service.WithInstancers(service.Instancers{"test": instancer}),
	)

	require.NotNil(e)

	instancer.On("Register", mock.AnythingOfType("chan<- sd.Event")).
		Run(func(arguments mock.Arguments) {
			registerQueue <- arguments.Get(0).(chan<- sd.Event)
		}).Once()

	instancer.On("Deregister", mock.AnythingOfType("chan<- sd.Event")).
		Run(func(arguments mock.Arguments) {
			registerQueue <- arguments.Get(0).(chan<- sd.Event)
		}).Once()

	instancer.On("Stop").Once()

	listener.On("MonitorEvent", mock.MatchedBy(func(Event) bool { return true })).Run(func(arguments mock.Arguments) {
		monitorEvents <- arguments.Get(0).(Event)
	})

	m, err := New(
		WithLogger(logger),
		WithFilter(nil),
		WithListeners(listener),
		WithEnvironment(e),
	)

	require.NoError(err)
	require.NotNil(m)

	select {
	case sdEvents = <-registerQueue:
	case <-time.After(5 * time.Second):
		m.Stop()
		require.Fail("Failed to receive registered event channel")
		return
	}

	sdEvents <- sd.Event{Err: expectedError}
	select {
	case event := <-monitorEvents:
		assert.Equal("test", event.Key)
		assert.Equal(expectedError, event.Err)
		assert.Len(event.Instances, 0)
		assert.False(event.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("Failed to receive monitor event")
	}

	sdEvents <- sd.Event{Instances: expectedInstances}
	select {
	case event := <-monitorEvents:
		assert.Equal("test", event.Key)
		assert.NoError(event.Err)
		assert.Equal(expectedInstances, event.Instances)
		assert.False(event.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("Failed to receive monitor event")
	}

	assert.NoError(e.Close())
	select {
	case deregistered := <-registerQueue:
		assert.Equal(sdEvents, deregistered)
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to deregister")
	}

	select {
	case finalEvent := <-monitorEvents:
		assert.Equal("test", finalEvent.Key)
		assert.NoError(finalEvent.Err)
		assert.Len(finalEvent.Instances, 0)
		assert.True(finalEvent.Stopped)

	case <-time.After(5 * time.Second):
		assert.Fail("No stopped event received")
	}

	select {
	case <-m.Stopped():
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to signal stopped channel")
	}

	// idempotency
	m.Stop()
	select {
	case <-m.Stopped():
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Failed to signal stopped channel")
	}

	instancer.AssertExpectations(t)
	listener.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Run("NoInstances", testNewNoInstances)
	t.Run("Stop", testNewStop)
	t.Run("WithEnvironment", testNewWithEnvironment)
}
