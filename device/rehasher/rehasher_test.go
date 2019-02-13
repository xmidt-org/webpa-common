package rehasher

import (
	"errors"
	"testing"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/monitor"
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

func TestNew(t *testing.T) {
	t.Run("NilConnector", testNewNilConnector)
	t.Run("MissingIsRegistered", testNewMissingIsRegistered)
}

func testRehasherServiceDiscoveryError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		serviceDiscoveryError = errors.New("service discovery error")
		connector             = new(device.MockConnector)
		rehasher              = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
		)
	)

	require.NotNil(rehasher)
	connector.On("DisconnectAll", device.CloseReason{Err: serviceDiscoveryError, Text: ServiceDiscoveryError}).Return(0)
	rehasher.MonitorEvent(monitor.Event{EventCount: 10, Err: serviceDiscoveryError})

	connector.AssertExpectations(t)
}

func testRehasherServiceDiscoveryStopped(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		rehasher  = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
		)
	)

	require.NotNil(rehasher)
	connector.On("DisconnectAll", device.CloseReason{Text: ServiceDiscoveryStopped}).Return(0)
	rehasher.MonitorEvent(monitor.Event{EventCount: 10, Stopped: true})

	connector.AssertExpectations(t)
}

func testRehasherInitialEvent(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		rehasher  = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
		)
	)

	require.NotNil(rehasher)
	rehasher.MonitorEvent(monitor.Event{EventCount: 1})

	connector.AssertExpectations(t)
}

func testRehasherNoInstances(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		isRegistered = func(string) bool {
			assert.Fail("isRegistered should not have been called")
			return false
		}

		connector = new(device.MockConnector)
		rehasher  = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
		)
	)

	require.NotNil(rehasher)
	connector.On("DisconnectAll", device.CloseReason{Text: ServiceDiscoveryNoInstances}).Return(0)
	rehasher.MonitorEvent(monitor.Event{EventCount: 10})

	connector.AssertExpectations(t)
}

func testRehasherRehash(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

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

		capture = make(chan func(device.ID) (device.CloseReason, bool), 1)

		connector = new(device.MockConnector)
		rehasher  = New(
			connector,
			WithLogger(logging.NewTestLogger(nil, t)),
			WithIsRegistered(isRegistered),
			WithAccessorFactory(accessorFactory),
		)
	)

	require.NotNil(rehasher)
	connector.On("DisconnectIf", mock.MatchedBy(
		func(func(device.ID) (device.CloseReason, bool)) bool { return true },
	)).
		Run(func(arguments mock.Arguments) {
			capture <- arguments.Get(0).(func(device.ID) (device.CloseReason, bool))
		}).
		Return(1)

	rehasher.MonitorEvent(monitor.Event{EventCount: 10, Instances: expectedNodes})

	select {
	case f := <-capture:
		reason, closed := f(keepID)
		assert.Equal(device.CloseReason{}, reason)
		assert.False(closed)

		reason, closed = f(rehashedID)
		assert.Equal(device.CloseReason{Text: RehashOtherInstance}, reason)
		assert.True(closed)

		reason, closed = f(accessorErrorID)
		assert.Equal(device.CloseReason{Err: accessorError, Text: RehashError}, reason)
		assert.True(closed)

	default:
		assert.Fail("No predicate captured: disconnection did not occur")
	}

	connector.AssertExpectations(t)
}

func TestRehasher(t *testing.T) {
	t.Run("ServiceDiscoveryError", testRehasherServiceDiscoveryError)
	t.Run("ServiceDiscoveryStopped", testRehasherServiceDiscoveryStopped)
	t.Run("InitialEvent", testRehasherInitialEvent)
	t.Run("NoInstances", testRehasherNoInstances)
	t.Run("Rehash", testRehasherRehash)
}
