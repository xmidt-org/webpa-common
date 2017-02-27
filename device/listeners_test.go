package device

import (
	"fmt"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultListeners(t *testing.T) {
	t.Log("smoke test for the internal default listeners")

	var (
		device   = new(mockDevice)
		encoded  = make([]byte, 10)
		message  = new(wrp.Message)
		pongData = "some lovely pong data!"
	)

	defaultMessageReceivedListener(device, message, encoded)
	defaultConnectListener(device)
	defaultDisconnectListener(device)
	defaultPongListener(device, pongData)

	device.AssertExpectations(t)
}

func TestMessageReceivedListeners(t *testing.T) {
	assert := assert.New(t)
	testData := [][]MessageReceivedListener{
		nil,
		make([]MessageReceivedListener, 0),
		make([]MessageReceivedListener, 1),
		make([]MessageReceivedListener, 2),
		make([]MessageReceivedListener, 5),
	}

	for _, listeners := range testData {
		var (
			expectedDevice  = new(mockDevice)
			expectedEncoded = []byte("test raw")
			expectedMessage = new(wrp.Message)
		)

		actualCallCount := 0
		for index, _ := range listeners {
			listeners[index] = func(actualDevice Interface, actualMessage *wrp.Message, actualEncoded []byte) {
				assert.True(expectedDevice == actualDevice)
				assert.Equal(expectedEncoded, actualEncoded)
				assert.True(expectedMessage == actualMessage)
				actualCallCount++
			}
		}

		messageListener := MessageReceivedListeners(listeners...)
		messageListener(expectedDevice, expectedMessage, expectedEncoded)

		assert.Equal(len(listeners), actualCallCount)
		expectedDevice.AssertExpectations(t)
	}
}

func TestConnectListeners(t *testing.T) {
	assert := assert.New(t)
	testData := [][]ConnectListener{
		nil,
		make([]ConnectListener, 0),
		make([]ConnectListener, 1),
		make([]ConnectListener, 2),
		make([]ConnectListener, 5),
	}

	for _, listeners := range testData {
		expectedDevice := new(mockDevice)

		actualCallCount := 0
		for index, _ := range listeners {
			listeners[index] = func(actualDevice Interface) {
				assert.True(expectedDevice == actualDevice)
				actualCallCount++
			}
		}

		connectListener := ConnectListeners(listeners...)
		connectListener(expectedDevice)

		assert.Equal(len(listeners), actualCallCount)
		expectedDevice.AssertExpectations(t)
	}
}

func TestDisconnectListeners(t *testing.T) {
	assert := assert.New(t)
	testData := [][]DisconnectListener{
		nil,
		make([]DisconnectListener, 0),
		make([]DisconnectListener, 1),
		make([]DisconnectListener, 2),
		make([]DisconnectListener, 5),
	}

	for _, listeners := range testData {
		expectedDevice := new(mockDevice)

		actualCallCount := 0
		for index, _ := range listeners {
			listeners[index] = func(actualDevice Interface) {
				assert.True(expectedDevice == actualDevice)
				actualCallCount++
			}
		}

		disconnectListener := DisconnectListeners(listeners...)
		disconnectListener(expectedDevice)

		assert.Equal(len(listeners), actualCallCount)
		expectedDevice.AssertExpectations(t)
	}
}

func TestPongListeners(t *testing.T) {
	assert := assert.New(t)
	testData := [][]PongListener{
		nil,
		make([]PongListener, 0),
		make([]PongListener, 1),
		make([]PongListener, 2),
		make([]PongListener, 5),
	}

	for index, listeners := range testData {
		expectedDevice := new(mockDevice)
		expectedPongData := fmt.Sprintf("pong data for iteration %d", index)

		actualCallCount := 0
		for index, _ := range listeners {
			listeners[index] = func(actualDevice Interface, actualPongData string) {
				assert.True(expectedDevice == actualDevice)
				assert.Equal(expectedPongData, actualPongData)
				actualCallCount++
			}
		}

		pongListener := PongListeners(listeners...)
		pongListener(expectedDevice, expectedPongData)

		assert.Equal(len(listeners), actualCallCount)
		expectedDevice.AssertExpectations(t)
	}
}
