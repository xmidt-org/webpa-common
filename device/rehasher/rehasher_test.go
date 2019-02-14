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

func testNewNilLogger(t *testing.T) {
	var (
		assert = assert.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(nil),
			WithIsRegistered(isRegistered),
		)
	)

	assert.NotNil(r)
	connector.AssertExpectations(t)
}

func testNewNilMetricsProvider(t *testing.T) {
	var (
		assert = assert.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithMetricsProvider(nil),
		)
	)

	assert.NotNil(r)
	connector.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Run("NilConnector", testNewNilConnector)
	t.Run("MissingIsRegistered", testNewMissingIsRegistered)
	t.Run("NilLogger", testNewNilLogger)
	t.Run("NilMetricsProvider", testNewNilMetricsProvider)
}

func testRehasherServiceDiscoveryError(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		serviceDiscoveryError = errors.New("service discovery error")
		connector             = new(device.MockConnector)
		r                     = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithMetricsProvider(provider),
		)
	)

	require.NotNil(r)
	connector.On("DisconnectAll", device.CloseReason{Err: serviceDiscoveryError, Text: ServiceDiscoveryError}).Return(12)
	provider.Expect(RehashKeepDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectAllCounter, service.ServiceLabel, "test", ReasonLabel, DisconnectAllServiceDiscoveryError)(
		xmetricstest.Counter,
		xmetricstest.Value(1.0),
	)
	provider.Expect(RehashTimestamp, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDurationMilliseconds, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))

	r.MonitorEvent(monitor.Event{EventCount: 10, Key: "test", Err: serviceDiscoveryError})

	connector.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func testRehasherServiceDiscoveryStopped(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithMetricsProvider(provider),
		)
	)

	require.NotNil(r)
	connector.On("DisconnectAll", device.CloseReason{Text: ServiceDiscoveryStopped}).Return(0)
	provider.Expect(RehashKeepDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectAllCounter, service.ServiceLabel, "test", ReasonLabel, DisconnectAllServiceDiscoveryStopped)(
		xmetricstest.Counter,
		xmetricstest.Value(1.0),
	)
	provider.Expect(RehashTimestamp, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDurationMilliseconds, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))

	r.MonitorEvent(monitor.Event{Key: "test", EventCount: 10, Stopped: true})

	connector.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func testRehasherInitialEvent(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithMetricsProvider(provider),
		)
	)

	require.NotNil(r)
	provider.Expect(RehashKeepDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectAllCounter, service.ServiceLabel, "test")(xmetricstest.Counter, xmetricstest.Value(0.0))
	provider.Expect(RehashTimestamp, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDurationMilliseconds, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))

	r.MonitorEvent(monitor.Event{Key: "test", EventCount: 1})

	connector.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func testRehasherNoInstances(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithMetricsProvider(provider),
		)
	)

	require.NotNil(r)
	connector.On("DisconnectAll", device.CloseReason{Text: ServiceDiscoveryNoInstances}).Return(0)
	provider.Expect(RehashKeepDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDisconnectAllCounter, service.ServiceLabel, "test", ReasonLabel, DisconnectAllServiceDiscoveryNoInstances)(
		xmetricstest.Counter,
		xmetricstest.Value(1.0),
	)
	provider.Expect(RehashTimestamp, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))
	provider.Expect(RehashDurationMilliseconds, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(0.0))

	r.MonitorEvent(monitor.Event{Key: "test", EventCount: 10})

	connector.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func testRehasherRehash(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		provider = xmetricstest.NewProvider(nil, Metrics)

		keepID   = device.ID("keep")
		keepNode = "keep.xfinity.net"

		rehashedID = device.ID("rehashed")
		rehashNode = "rehash.xfinity.net"

		accessorErrorID = device.ID("accessorError")
		accessorError   = errors.New("expected accessor error")

		expectedNodes   = []string{keepNode, rehashNode}
		accessorFactory = service.AccessorFactory(func(actualNodes []string) service.Accessor {
			assert.Equal(expectedNodes, actualNodes)
			return service.AccessorFunc(func(key []byte) (string, error) {
				switch string(key) {
				case string(keepID):
					return keepNode, nil

				case string(rehashedID):
					return rehashNode, nil

				case string(accessorErrorID):
					return "", accessorError

				default:
					assert.Fail("Invalid accessor key")
					return "", errors.New("test failure: invalid accessor key")
				}
			})
		})

		isRegistered = func(v string) bool {
			return keepNode == v
		}

		expectedDuration = 10 * time.Minute
		start            = time.Now()
		end              = start.Add(expectedDuration)
		nowFirst         = true
		now              = func() time.Time {
			if nowFirst {
				nowFirst = false
				return start
			}

			return end
		}

		connector = new(device.MockConnector)
		r         = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithAccessorFactory(accessorFactory),
			WithMetricsProvider(provider),
		)
	)

	require.NotNil(r)
	r.(*rehasher).now = now
	connector.On("DisconnectIf", mock.MatchedBy(
		func(func(device.ID) (device.CloseReason, bool)) bool { return true },
	)).
		Run(func(arguments mock.Arguments) {
			f := arguments.Get(0).(func(device.ID) (device.CloseReason, bool))

			reason, closed := f(keepID)
			assert.Equal(device.CloseReason{}, reason)
			assert.False(closed)

			reason, closed = f(rehashedID)
			assert.Equal(device.CloseReason{Text: RehashOtherInstance}, reason)
			assert.True(closed)

			reason, closed = f(accessorErrorID)
			assert.Equal(device.CloseReason{Err: accessorError, Text: RehashError}, reason)
			assert.True(closed)
		}).
		Return(2)

	provider.Expect(RehashKeepDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(1.0))
	provider.Expect(RehashDisconnectDevice, service.ServiceLabel, "test")(xmetricstest.Gauge, xmetricstest.Value(2.0))
	provider.Expect(RehashDisconnectAllCounter, service.ServiceLabel, "test")(xmetricstest.Counter, xmetricstest.Value(0.0))
	provider.Expect(RehashTimestamp, service.ServiceLabel, "test")(
		xmetricstest.Gauge,
		xmetricstest.Value(float64(start.UTC().Unix())),
	)
	provider.Expect(RehashDurationMilliseconds, service.ServiceLabel, "test")(
		xmetricstest.Gauge,
		xmetricstest.Value(float64(expectedDuration/time.Millisecond)),
	)

	r.MonitorEvent(monitor.Event{Key: "test", EventCount: 10, Instances: expectedNodes})

	connector.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func TestRehasher(t *testing.T) {
	t.Run("ServiceDiscoveryError", testRehasherServiceDiscoveryError)
	t.Run("ServiceDiscoveryStopped", testRehasherServiceDiscoveryStopped)
	t.Run("InitialEvent", testRehasherInitialEvent)
	t.Run("NoInstances", testRehasherNoInstances)
	t.Run("Rehash", testRehasherRehash)
}
