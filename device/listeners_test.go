package device

import (
	"fmt"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestDefaultListener(t *testing.T) {
	t.Log("smoke test for the internal default listener")
	device := new(mockDevice)
	message := new(wrp.Message)
	pongData := "TestDefaultListener"

	defaultListener.OnMessage(device, message)
	defaultListener.OnConnect(device)
	defaultListener.OnDisconnect(device)
	defaultListener.OnPong(device, pongData)

	device.AssertExpectations(t)
}

func TestMessageListeners(t *testing.T) {
	testData := []MessageListeners{
		nil,
		make(MessageListeners, 0),
		make(MessageListeners, 1),
		make(MessageListeners, 2),
		make(MessageListeners, 5),
	}

	for _, listeners := range testData {
		device := new(mockDevice)
		deviceMatcher := mock.MatchedBy(func(d Interface) bool { return d == device })

		message := new(wrp.Message)
		messageMatcher := mock.MatchedBy(func(m *wrp.Message) bool { return m == message })

		for index := 0; index < len(listeners); index++ {
			mockDeviceListener := new(mockDeviceListener)
			mockDeviceListener.On("OnMessage", deviceMatcher, messageMatcher).Once()
			listeners[index] = mockDeviceListener
		}

		listeners.OnMessage(device, message)

		for _, listener := range listeners {
			listener.(*mockDeviceListener).AssertExpectations(t)
		}

		device.AssertExpectations(t)
	}
}

func TestConnectListeners(t *testing.T) {
	testData := []ConnectListeners{
		nil,
		make(ConnectListeners, 0),
		make(ConnectListeners, 1),
		make(ConnectListeners, 2),
		make(ConnectListeners, 5),
	}

	for _, listeners := range testData {
		device := new(mockDevice)
		deviceMatcher := mock.MatchedBy(func(d Interface) bool { return d == device })

		for index := 0; index < len(listeners); index++ {
			mockDeviceListener := new(mockDeviceListener)
			mockDeviceListener.On("OnConnect", deviceMatcher).Once()
			listeners[index] = mockDeviceListener
		}

		listeners.OnConnect(device)

		for _, listener := range listeners {
			listener.(*mockDeviceListener).AssertExpectations(t)
		}

		device.AssertExpectations(t)
	}
}

func TestDisconnectListeners(t *testing.T) {
	testData := []DisconnectListeners{
		nil,
		make(DisconnectListeners, 0),
		make(DisconnectListeners, 1),
		make(DisconnectListeners, 2),
		make(DisconnectListeners, 5),
	}

	for _, listeners := range testData {
		device := new(mockDevice)
		deviceMatcher := mock.MatchedBy(func(d Interface) bool { return d == device })

		for index := 0; index < len(listeners); index++ {
			mockDeviceListener := new(mockDeviceListener)
			mockDeviceListener.On("OnDisconnect", deviceMatcher).Once()
			listeners[index] = mockDeviceListener
		}

		listeners.OnDisconnect(device)

		for _, listener := range listeners {
			listener.(*mockDeviceListener).AssertExpectations(t)
		}

		device.AssertExpectations(t)
	}
}

func TestPongListeners(t *testing.T) {
	testData := []PongListeners{
		nil,
		make(PongListeners, 0),
		make(PongListeners, 1),
		make(PongListeners, 2),
		make(PongListeners, 5),
	}

	for index, listeners := range testData {
		device := new(mockDevice)
		deviceMatcher := mock.MatchedBy(func(d Interface) bool { return d == device })

		pongData := fmt.Sprintf("pong data %d", index)

		for index := 0; index < len(listeners); index++ {
			mockDeviceListener := new(mockDeviceListener)
			mockDeviceListener.On("OnPong", deviceMatcher, pongData).Once()
			listeners[index] = mockDeviceListener
		}

		listeners.OnPong(device, pongData)

		for _, listener := range listeners {
			listener.(*mockDeviceListener).AssertExpectations(t)
		}

		device.AssertExpectations(t)
	}
}
