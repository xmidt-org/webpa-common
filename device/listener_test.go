package device

import (
	"errors"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testEventString(t *testing.T) {
	var (
		assert     = assert.New(t)
		values     = make(map[string]bool)
		eventTypes = []EventType{
			Connect,
			Disconnect,
			MessageReceived,
			MessageFailed,
			Pong,
		}
	)

	for _, eventType := range eventTypes {
		value := eventType.String()
		assert.NotEqual(InvalidEventString, value)
		assert.NotContains(values, value)
		values[value] = true
	}

	assert.Equal(InvalidEventString, EventType(255).String())
}

func testEventSetMessageFailedWithError(t *testing.T, event Event) {
	var (
		assert  = assert.New(t)
		device  = new(mockDevice)
		message = new(wrp.Message)
		encoded = []byte("testEventSetMessageFailed")
		err     = errors.New("testEventSetMessageFailed")
	)

	event.setMessageFailed(device, message, encoded, err)
	assert.Equal(MessageFailed, event.Type)
	assert.Equal(device, event.Device)
	assert.True(message == event.Message)
	assert.Equal(encoded, event.Encoded)
	assert.True(err == event.Err)
	assert.Empty(event.Data)

	device.AssertExpectations(t)
}

func testEventSetMessageFailedWithoutError(t *testing.T, event Event) {
	var (
		assert  = assert.New(t)
		device  = new(mockDevice)
		message = new(wrp.Message)
		encoded = []byte("testEventSetMessageFailed")
	)

	event.setMessageFailed(device, message, encoded, nil)
	assert.Equal(MessageFailed, event.Type)
	assert.Equal(device, event.Device)
	assert.True(message == event.Message)
	assert.Equal(encoded, event.Encoded)
	assert.NoError(event.Err)
	assert.Empty(event.Data)

	device.AssertExpectations(t)
}

func testEventSetMessageReceived(t *testing.T, event Event) {
	var (
		assert  = assert.New(t)
		device  = new(mockDevice)
		message = new(wrp.Message)
		encoded = []byte("testEventSetMessageReceived")
	)

	event.setMessageReceived(device, message, encoded)
	assert.Equal(MessageReceived, event.Type)
	assert.Equal(device, event.Device)
	assert.True(message == event.Message)
	assert.Equal(encoded, event.Encoded)
	assert.NoError(event.Err)
	assert.Empty(event.Data)
	device.AssertExpectations(t)
}

func testEventSetPong(t *testing.T, event Event) {
	var (
		assert = assert.New(t)
		device = new(mockDevice)
		data   = "testSetPong"
	)

	event.setPong(device, data)
	assert.Equal(Pong, event.Type)
	assert.Equal(device, event.Device)
	assert.Nil(event.Message)
	assert.Empty(event.Encoded)
	assert.NoError(event.Err)
	assert.Equal(data, event.Data)
	device.AssertExpectations(t)
}

func TestEvent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		testEventString(t)
	})

	var (
		device = new(mockDevice)
		events = []Event{
			Event{},
			Event{
				Type:   Connect,
				Device: device,
			},
			Event{
				Type:   Disconnect,
				Device: device,
			},
			Event{
				Type:    MessageFailed,
				Device:  device,
				Message: new(wrp.Message),
				Encoded: []byte("encoded"),
			},
			Event{
				Type:    MessageFailed,
				Device:  device,
				Message: new(wrp.Message),
				Encoded: []byte("encoded"),
				Err:     errors.New("some random I/O problem"),
			},
			Event{
				Type:    MessageReceived,
				Device:  device,
				Message: new(wrp.Message),
				Encoded: []byte("encoded"),
			},
			Event{
				Type:   Pong,
				Device: device,
				Data:   "some pong data",
			},
		}
	)

	t.Run("setMessageFailed", func(t *testing.T) {
		for _, original := range events {
			testEventSetMessageFailedWithError(t, original)
			testEventSetMessageFailedWithoutError(t, original)
		}
	})

	t.Run("setMessageReceived", func(t *testing.T) {
		for _, original := range events {
			testEventSetMessageReceived(t, original)
		}
	})

	t.Run("setPong", func(t *testing.T) {
		for _, original := range events {
			testEventSetPong(t, original)
		}
	})

	device.AssertExpectations(t)
}
