package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenerFunc(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedEvent = Event{Instances: []string{"instance1"}}

		called = false
		lf     = func(actualEvent Event) {
			called = true
			assert.Equal(expectedEvent, actualEvent)
		}
	)

	ListenerFunc(lf).MonitorEvent(expectedEvent)
	assert.True(called)
}

func testListenersEmpty(t *testing.T, l Listeners) {
	assert := assert.New(t)

	assert.NotPanics(func() {
		l.MonitorEvent(Event{})
	})
}

func testListenersNonEmpty(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedEvent = Event{Instances: []string{"foobar.com", "shaky.net"}}
		firstCalled   = false
		secondCalled  = false

		l = Listeners{
			ListenerFunc(func(actualEvent Event) {
				firstCalled = true
				assert.Equal(expectedEvent, actualEvent)
			}),
			ListenerFunc(func(actualEvent Event) {
				secondCalled = true
				assert.Equal(expectedEvent, actualEvent)
			}),
		}
	)

	l.MonitorEvent(expectedEvent)
	assert.True(firstCalled)
	assert.True(secondCalled)
}

func TestListeners(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		testListenersEmpty(t, nil)
		testListenersEmpty(t, Listeners{})
	})

	t.Run("NonEmpty", testListenersNonEmpty)
}

func testNewMetricsListenerUpdate(t *testing.T) {
	var (
		now = float64(time.Now().Unix())

		p = xmetricstest.NewProvider(nil, service.Metrics).
			Expect(service.UpdateCount, service.ServiceLabel, "test")(xmetricstest.Value(1.0)).
			Expect(service.LastUpdateTimestamp, service.ServiceLabel, "test")(xmetricstest.Minimum(now)).
			Expect(service.ErrorCount, service.ServiceLabel, "test")(xmetricstest.Value(0.0)).
			Expect(service.LastErrorTimestamp, service.ServiceLabel, "test")(xmetricstest.Value(0.0)).
			Expect(service.InstanceCount, service.ServiceLabel, "test")(xmetricstest.Value(2.0))
		l = NewMetricsListener(p)
	)

	l.MonitorEvent(Event{Key: "test", Instances: []string{"instance1", "instance2"}})
	p.AssertExpectations(t)
}

func testNewMetricsListenerError(t *testing.T) {
	var (
		now = float64(time.Now().Unix())

		p = xmetricstest.NewProvider(nil, service.Metrics).
			Expect(service.UpdateCount, service.ServiceLabel, "test")(xmetricstest.Value(0.0)).
			Expect(service.LastUpdateTimestamp, service.ServiceLabel, "test")(xmetricstest.Value(0.0)).
			Expect(service.ErrorCount, service.ServiceLabel, "test")(xmetricstest.Value(1.0)).
			Expect(service.LastErrorTimestamp, service.ServiceLabel, "test")(xmetricstest.Minimum(now)).
			Expect(service.InstanceCount, service.ServiceLabel, "test")(xmetricstest.Value(0.0))
		l = NewMetricsListener(p)
	)

	l.MonitorEvent(Event{Key: "test", Err: errors.New("expected")})
	p.AssertExpectations(t)
}

func TestNewMetricsListener(t *testing.T) {
	t.Run("Update", testNewMetricsListenerUpdate)
	t.Run("Error", testNewMetricsListenerError)
}

func testNewAccessorListenerMissingNext(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewAccessorListener(service.DefaultAccessorFactory, nil)
	})
}

func testNewAccessorListenerError(t *testing.T, f service.AccessorFactory) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedError = errors.New("expected")
		nextCalled    = false

		l = NewAccessorListener(
			f,
			func(a service.Accessor, err error) {
				nextCalled = true
				assert.Nil(a)
				assert.Equal(expectedError, err)
			},
		)
	)

	require.NotNil(l)
	l.MonitorEvent(Event{Err: expectedError})
	assert.True(nextCalled)
}

func testNewAccessorListenerInstances(t *testing.T, f service.AccessorFactory) {
	var (
		assert     = assert.New(t)
		require    = require.New(t)
		nextCalled = false

		l = NewAccessorListener(
			nil,
			func(a service.Accessor, err error) {
				nextCalled = true
				require.NotNil(a)
				assert.NoError(err)

				i, err := a.Get([]byte("asdfasdfasdfsdf"))
				assert.Equal("instance1", i)
				assert.NoError(err)
			},
		)
	)

	require.NotNil(l)
	l.MonitorEvent(Event{Instances: []string{"instance1"}})
	assert.True(nextCalled)
}

func testNewAccessorListenerEmpty(t *testing.T, f service.AccessorFactory) {
	var (
		assert     = assert.New(t)
		require    = require.New(t)
		nextCalled = false

		l = NewAccessorListener(
			nil,
			func(a service.Accessor, err error) {
				nextCalled = true
				assert.Equal(service.EmptyAccessor(), a)
				assert.NoError(err)
			},
		)
	)

	require.NotNil(l)
	l.MonitorEvent(Event{})
	assert.True(nextCalled)
}

func TestNewAccessorListener(t *testing.T) {
	t.Run("MissingNext", testNewAccessorListenerMissingNext)

	t.Run("DefaultAccessorFactory", func(t *testing.T) {
		t.Run("Error", func(t *testing.T) {
			testNewAccessorListenerError(t, service.DefaultAccessorFactory)
		})

		t.Run("Instances", func(t *testing.T) {
			testNewAccessorListenerInstances(t, service.DefaultAccessorFactory)
		})

		t.Run("Empty", func(t *testing.T) {
			testNewAccessorListenerEmpty(t, service.DefaultAccessorFactory)
		})
	})

	t.Run("CustomAccessorFactory", func(t *testing.T) {
		f := service.NewConsistentAccessorFactory(2)

		t.Run("Error", func(t *testing.T) {
			testNewAccessorListenerError(t, f)
		})

		t.Run("Instances", func(t *testing.T) {
			testNewAccessorListenerInstances(t, f)
		})

		t.Run("Empty", func(t *testing.T) {
			testNewAccessorListenerEmpty(t, f)
		})
	})
}
