package device

import (
	"errors"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
)

func testEventString(t *testing.T) {
	var (
		assert     = assert.New(t)
		values     = make(map[string]bool)
		eventTypes = []EventType{
			Connect,
			Disconnect,
			MessageSent,
			MessageReceived,
			MessageFailed,
			TransactionComplete,
			TransactionBroken,
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

func testEventClear(t *testing.T, event Event) {
	assert := assert.New(t)

	event.Clear()
	assert.Equal(EventType(0), event.Type)
	assert.Nil(event.Device)
	assert.Nil(event.Message)
	assert.Equal(wrp.Msgpack, event.Format)
	assert.Nil(event.Contents)
	assert.Nil(event.Error)
	assert.Empty(event.Data)
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
				Type:     MessageFailed,
				Device:   device,
				Message:  new(wrp.Message),
				Contents: []byte("contents"),
			},
			Event{
				Type:     MessageFailed,
				Device:   device,
				Message:  new(wrp.Message),
				Contents: []byte("contents"),
				Error:    errors.New("some random I/O problem"),
			},
			Event{
				Type:     MessageReceived,
				Device:   device,
				Message:  new(wrp.Message),
				Contents: []byte("contents"),
			},
			Event{
				Type:   Pong,
				Device: device,
				Data:   "some pong data",
			},
		}
	)

	t.Run("Clear", func(t *testing.T) {
		for _, original := range events {
			testEventClear(t, original)
		}
	})

	device.AssertExpectations(t)
}
