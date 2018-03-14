package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
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
