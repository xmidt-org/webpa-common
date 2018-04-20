package rehasher

import (
	"errors"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/monitor"
	"github.com/Comcast/webpa-common/xmetrics/xmetricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testNewNilConnector(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		New(nil, WithAccessorFactory(nil), WithIsRegistered(func(string) bool { return true }))
	})
}

func testNewMissingIsRegistered(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = new(device.MockConnector)
	)

	assert.Panics(func() {
		New(c, WithAccessorFactory(nil))
	})

	c.AssertExpectations(t)
}

func testNewWithIsRegistered(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		c = new(device.MockConnector)
		e = new(service.MockEnvironment)
		a = new(service.MockAccessor)

		i                   = new(service.MockInstancer)
		contextualInstancer = service.NewContextualInstancer(
			i,
			map[string]interface{}{"server": "localhost:8000"},
		)

		errorID          = device.ID("error")
		keepID           = device.ID("keep")
		disconnectID     = device.ID("disconnect")
		predicateCapture = make(chan func(device.ID) bool, 1)
	)

	a.On("Get", errorID.Bytes()).Return("", errors.New("expected")).Once()
	a.On("Get", keepID.Bytes()).Return("keep", error(nil)).Once()
	a.On("Get", disconnectID.Bytes()).Return("disconnect", error(nil)).Once()

	e.On("IsRegistered", "keep").Return(true)
	e.On("IsRegistered", "disconnect").Return(false)

	c.On("DisconnectAll").Return(0).Times(3)
	c.On("DisconnectIf", mock.AnythingOfType("func(device.ID) bool")).Return(1).Once().
		Run(func(arguments mock.Arguments) {
			predicateCapture <- arguments.Get(0).(func(device.ID) bool)
		})

	l := New(c, WithLogger(nil), WithAccessorFactory(func([]string) service.Accessor { return a }), WithIsRegistered(e.IsRegistered), WithMetricsProvider(nil))
	require.NotNil(l)

	l.MonitorEvent(monitor.Event{Key: "testNewWithIsRegistered", Instancer: contextualInstancer, EventCount: 1})
	l.MonitorEvent(monitor.Event{Key: "testNewWithIsRegistered", Instancer: contextualInstancer, EventCount: 2, Err: errors.New("service discovery error")})
	l.MonitorEvent(monitor.Event{Key: "testNewWithIsRegistered", Instancer: contextualInstancer, EventCount: 2, Stopped: true})
	l.MonitorEvent(monitor.Event{})
	l.MonitorEvent(monitor.Event{Key: "testNewWithIsRegistered", Instancer: contextualInstancer, EventCount: 4, Instances: []string{"keep", "disconnect"}})

	select {
	case predicate := <-predicateCapture:
		assert.True(predicate(errorID))
		assert.False(predicate(keepID))
		assert.True(predicate(disconnectID))
	case <-time.After(time.Second):
		require.Fail("No predicate sent")
	}

	a.AssertExpectations(t)
	c.AssertExpectations(t)
	e.AssertExpectations(t)
	i.AssertExpectations(t)
}

func testNewWithEnvironment(t *testing.T) {
	const key = "testNewWithEnvironment"

	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		c  = new(device.MockConnector)
		r  = new(device.MockRegistry)
		e  = new(service.MockEnvironment)
		a  = new(service.MockAccessor)
		af = service.AccessorFactory(func([]string) service.Accessor {
			return a
		})

		i                   = new(service.MockInstancer)
		contextualInstancer = service.NewContextualInstancer(
			i,
			map[string]interface{}{"server": "localhost:8000"},
		)

		errorID      = device.ID("error")
		keepID       = device.ID("keep")
		disconnectID = device.ID("disconnect")

		expectedStart = time.Now()
		expectedEnd   = expectedStart.Add(10 * time.Second)
		started       = false
		now           = func() time.Time {
			if started {
				return expectedEnd
			}

			started = true
			return expectedStart
		}
	)

	a.On("Get", errorID.Bytes()).Return("", errors.New("expected")).Once()
	a.On("Get", keepID.Bytes()).Return("keep", error(nil)).Once()
	a.On("Get", disconnectID.Bytes()).Return("disconnect", error(nil)).Once()

	e.On("AccessorFactory").Return(af).Once()
	e.On("IsRegistered", "keep").Return(true)
	e.On("IsRegistered", "disconnect").Return(false)

	c.On("DisconnectAll").Return(0).Times(3)
	c.On("DisconnectIf", mock.AnythingOfType("func(device.ID) bool")).Return(1).Once().
		Run(func(arguments mock.Arguments) {
			f := arguments.Get(0).(func(device.ID) bool)
			assert.True(f(errorID))
			assert.False(f(keepID))
			assert.True(f(disconnectID))
		})

	l := New(c, WithLogger(logging.NewTestLogger(nil, t)), WithEnvironment(e), WithMetricsProvider(provider))
	require.NotNil(l)
	l.(*rehasher).now = now

	// this will be ignored, as the first event
	l.MonitorEvent(monitor.Event{Key: key, Instancer: contextualInstancer, EventCount: 1})
	provider.Assert(t, RehashKeepDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryError)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryStopped)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashTimestamp, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDurationMilliseconds, service.ServiceLabel, key)(xmetricstest.Value(0.0))

	l.MonitorEvent(monitor.Event{Key: key, Instancer: contextualInstancer, EventCount: 2, Err: errors.New("service discovery error")})
	provider.Assert(t, RehashKeepDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryError)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryStopped)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashTimestamp, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDurationMilliseconds, service.ServiceLabel, key)(xmetricstest.Value(0.0))

	l.MonitorEvent(monitor.Event{Key: key, Instancer: contextualInstancer, EventCount: 2, Stopped: true})
	provider.Assert(t, RehashKeepDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryError)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryStopped)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashTimestamp, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDurationMilliseconds, service.ServiceLabel, key)(xmetricstest.Value(0.0))

	l.MonitorEvent(monitor.Event{Key: key, EventCount: 3})
	provider.Assert(t, RehashKeepDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectDevice, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryError)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryStopped)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashTimestamp, service.ServiceLabel, key)(xmetricstest.Value(0.0))
	provider.Assert(t, RehashDurationMilliseconds, service.ServiceLabel, key)(xmetricstest.Value(0.0))

	l.MonitorEvent(monitor.Event{Key: key, Instancer: contextualInstancer, EventCount: 4, Instances: []string{"keep", "disconnect"}})
	provider.Assert(t, RehashKeepDevice, service.ServiceLabel, key)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectDevice, service.ServiceLabel, key)(xmetricstest.Value(1.0)) // uses the return value of DisconnectIf
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryError)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashDisconnectAllCounter, service.ServiceLabel, key, ReasonLabel, DisconnectAllServiceDiscoveryStopped)(xmetricstest.Value(1.0))
	provider.Assert(t, RehashTimestamp, service.ServiceLabel, key)(xmetricstest.Value(float64(expectedStart.UTC().Unix())))
	provider.Assert(t, RehashDurationMilliseconds, service.ServiceLabel, key)(xmetricstest.Value(float64(10 * time.Second / time.Millisecond)))

	a.AssertExpectations(t)
	c.AssertExpectations(t)
	r.AssertExpectations(t)
	e.AssertExpectations(t)
	i.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Run("NilConnector", testNewNilConnector)
	t.Run("MissingIsRegistered", testNewMissingIsRegistered)
	t.Run("WithIsRegistered", testNewWithIsRegistered)
	t.Run("WithEnvironment", testNewWithEnvironment)
}
