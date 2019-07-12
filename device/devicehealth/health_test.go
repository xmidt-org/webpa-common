package devicehealth

import (
	"testing"

	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testListenerOnDeviceEventConnect(t *testing.T) {
	var (
		assert     = assert.New(t)
		dispatcher = new(mockDispatcher)
		listener   = &Listener{Dispatcher: dispatcher}

		expectedStats = health.Stats{
			DeviceCount:           1,
			TotalConnectionEvents: 1,
		}

		actualStats = health.Stats{}
	)

	dispatcher.On("SendEvent", mock.AnythingOfType("health.HealthFunc")).Once().
		Run(func(arguments mock.Arguments) {
			hf := arguments.Get(0).(health.HealthFunc)
			hf(actualStats)
		})

	listener.OnDeviceEvent(&device.Event{Type: device.Connect})
	assert.Equal(expectedStats, actualStats)

	dispatcher.AssertExpectations(t)
}

func testListenerOnDeviceEventDisconnect(t *testing.T) {
	var (
		assert     = assert.New(t)
		dispatcher = new(mockDispatcher)
		listener   = &Listener{Dispatcher: dispatcher}

		expectedStats = health.Stats{
			DeviceCount:              0,
			TotalConnectionEvents:    1,
			TotalDisconnectionEvents: 1,
		}

		actualStats = health.Stats{
			DeviceCount:           1,
			TotalConnectionEvents: 1,
		}
	)

	dispatcher.On("SendEvent", mock.AnythingOfType("health.HealthFunc")).Once().
		Run(func(arguments mock.Arguments) {
			hf := arguments.Get(0).(health.HealthFunc)
			hf(actualStats)
		})

	listener.OnDeviceEvent(&device.Event{Type: device.Disconnect})
	assert.Equal(expectedStats, actualStats)

	dispatcher.AssertExpectations(t)
}

func testListenerOnDeviceEventTransactionComplete(t *testing.T) {
	var (
		assert     = assert.New(t)
		dispatcher = new(mockDispatcher)
		listener   = &Listener{Dispatcher: dispatcher}

		expectedStats = health.Stats{
			TotalWRPRequestResponseProcessed: 1,
		}

		actualStats = health.Stats{}
	)

	dispatcher.On("SendEvent", mock.AnythingOfType("health.HealthFunc")).Once().
		Run(func(arguments mock.Arguments) {
			hf := arguments.Get(0).(health.HealthFunc)
			hf(actualStats)
		})

	listener.OnDeviceEvent(&device.Event{Type: device.TransactionComplete})
	assert.Equal(expectedStats, actualStats)

	dispatcher.AssertExpectations(t)
}

func TestListener(t *testing.T) {
	t.Run("OnDeviceEvent", func(t *testing.T) {
		t.Run("Connect", testListenerOnDeviceEventConnect)
		t.Run("Disconnect", testListenerOnDeviceEventDisconnect)
		t.Run("TransactionComplete", testListenerOnDeviceEventTransactionComplete)
	})
}
