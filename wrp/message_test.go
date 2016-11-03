package wrp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageTypeString(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		messageType    MessageType
		expectedString string
	}{
		{MessageType(0), invalidMessageTypeString},
		{MessageType(1), invalidMessageTypeString},
		{AuthMessageType, messageTypeStrings[AuthMessageType]},
		{SimpleRequestResponseMessageType, messageTypeStrings[SimpleRequestResponseMessageType]},
		{SimpleEventMessageType, messageTypeStrings[SimpleEventMessageType]},
		{MessageType(999), invalidMessageTypeString},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedString, record.messageType.String())
	}
}

func TestMessageValid(t *testing.T) {
	assert := assert.New(t)
	expectedStatus := int64(987)

	var testData = []struct {
		message       Message
		expectedValid bool
	}{
		{
			Message{},
			false,
		},
		{
			Message{
				Type: MessageType(12345),
			},
			false,
		},
		{
			Message{
				Type: AuthMessageType,
			},
			false,
		},
		{
			Message{
				Type:   AuthMessageType,
				Status: &expectedStatus,
			},
			true,
		},
		{
			Message{
				Type: SimpleRequestResponseMessageType,
			},
			false,
		},
		{
			Message{
				Type:        SimpleRequestResponseMessageType,
				Destination: "dns:foobar.com",
			},
			true,
		},
		{
			Message{
				Type: SimpleEventMessageType,
			},
			false,
		},
		{
			Message{
				Type:        SimpleEventMessageType,
				Destination: "dns:foobar.com",
			},
			true,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedValid, record.message.Valid() == nil)
	}
}

func TestMessageNilString(t *testing.T) {
	assert := assert.New(t)
	var message *Message
	assert.Equal("nil", message.String())
}

func TestMessageDeduceType(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		original     Message
		expectedType MessageType
		expectsError bool
	}{
		{
			Message{},
			MessageType(0),
			true,
		},
		{
			Message{Type: MessageType(999)},
			MessageType(999),
			true,
		},
		{
			Message{
				Status: &expectedStatus,
			},
			AuthMessageType,
			false,
		},
		{
			Message{
				Destination: "foobar.com",
			},
			SimpleEventMessageType,
			false,
		},
		{
			Message{
				Source:      "serial:1234",
				Destination: "foobar.com",
			},
			SimpleRequestResponseMessageType,
			false,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)

		message := record.original
		assert.Equal(record.expectsError, message.DeduceType() != nil)
		assert.Equal(record.expectedType, message.Type)
	}
}

func TestNewAuth(t *testing.T) {
	assert := assert.New(t)

	for _, expectedStatus := range []int64{-1234, 0, 1, 2, 239457} {
		t.Logf("%d", expectedStatus)
		message := NewAuth(expectedStatus)
		if assert.NotNil(message) {
			assert.Equal(AuthMessageType, message.Type)
			assert.Equal(expectedStatus, *message.Status)
			assert.Empty(message.TransactionUUID)
			assert.Empty(message.Destination)
			assert.Empty(message.Source)
			assert.Empty(message.Payload)
		}
	}
}

func TestNewSimpleRequestResponse(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedDestination string
		expectedSource      string
		expectedPayload     []byte
	}{
		{"foobar.com", "112233445566", nil},
		{"test.com/bleh", "FFFFEEEEDDDD", []byte("hi there!")},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewSimpleRequestResponse(record.expectedDestination, record.expectedSource, record.expectedPayload)
		if assert.NotNil(message) {
			assert.Equal(SimpleRequestResponseMessageType, message.Type)
			assert.Nil(message.Status)
			assert.Empty(message.TransactionUUID)
			assert.Equal(record.expectedDestination, message.Destination)
			assert.Equal(record.expectedSource, message.Source)
			assert.Equal(record.expectedPayload, message.Payload)
		}
	}
}

func TestNewSimpleEvent(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		expectedDestination string
		expectedPayload     []byte
	}{
		{"foobar.com", nil},
		{"test.com/bleh", []byte("hi there!")},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		message := NewSimpleEvent(record.expectedDestination, record.expectedPayload)
		if assert.NotNil(message) {
			assert.Equal(SimpleEventMessageType, message.Type)
			assert.Nil(message.Status)
			assert.Empty(message.TransactionUUID)
			assert.Equal(record.expectedDestination, message.Destination)
			assert.Empty(message.Source)
			assert.Equal(record.expectedPayload, message.Payload)
		}
	}
}
