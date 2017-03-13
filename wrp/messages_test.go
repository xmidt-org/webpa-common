package wrp

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	// allFormats enumerates all of the supported formats to use in testing
	allFormats = []Format{JSON, Msgpack}
)

func TestMessageTypeString(t *testing.T) {
	var (
		assert       = assert.New(t)
		messageTypes = []MessageType{
			AuthMessageType,
			SimpleRequestResponseMessageType,
			SimpleEventMessageType,
			CreateMessageType,
			RetrieveMessageType,
			UpdateMessageType,
			DeleteMessageType,
			ServiceRegistrationMessageType,
			ServiceAliveMessageType,
		}

		strings = make(map[string]bool, len(messageTypes))
	)

	for _, messageType := range messageTypes {
		stringValue := messageType.String()
		assert.NotEmpty(stringValue)

		assert.NotContains(strings, stringValue)
		strings[stringValue] = true
	}

	assert.Equal(len(messageTypes), len(strings))
	assert.Equal(InvalidMessageTypeString, MessageType(-1).String())
	assert.NotContains(strings, InvalidMessageTypeString)
}

func testMessageSetStatus(t *testing.T) {
	var (
		assert  = assert.New(t)
		message Message
	)

	assert.Nil(message.Status)
	assert.True(&message == message.SetStatus(72))
	assert.NotNil(message.Status)
	assert.Equal(int64(72), *message.Status)
	assert.True(&message == message.SetStatus(6))
	assert.NotNil(message.Status)
	assert.Equal(int64(6), *message.Status)
}

func testMessageSetRequestDeliveryResponse(t *testing.T) {
	var (
		assert  = assert.New(t)
		message Message
	)

	assert.Nil(message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(14))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(14), *message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(456))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(456), *message.RequestDeliveryResponse)
}

func testMessageSetIncludeSpans(t *testing.T) {
	var (
		assert  = assert.New(t)
		message Message
	)

	assert.Nil(message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(true))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(true, *message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(false))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(false, *message.IncludeSpans)
}

func testMessageRoutable(t *testing.T, original Message) {
	assert := assert.New(t)
	assert.Equal(original.Type, original.MessageType())
	assert.Equal(original.Destination, original.To())
	assert.Equal(original.Source, original.From())
}

func testMessageEncode(t *testing.T, f Format, original Message) {
	var (
		assert  = assert.New(t)
		decoded Message

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func testMessageFailureResponse(t *testing.T, original Message) {
	const (
		expectedSource                        = "testMessageFailureResponse"
		expectedRequestDeliveryResponse int64 = 65873
	)

	var (
		assert        = assert.New(t)
		require       = require.New(t)
		checkResponse = func(actual *Message) {
			// these fields should have been changed in some way
			assert.Equal(original.Source, actual.Destination)
			assert.Equal(expectedSource, actual.Source)
			require.NotNil(actual.RequestDeliveryResponse)
			assert.Equal(expectedRequestDeliveryResponse, *actual.RequestDeliveryResponse)
			assert.Nil(actual.Payload)

			// these fields should be the same
			assert.Equal(original.Type, actual.Type)
			assert.Equal(original.TransactionUUID, actual.TransactionUUID)
			assert.Equal(original.ContentType, actual.ContentType)
			assert.Equal(original.Accept, actual.Accept)
			assert.Equal(original.Status, actual.Status)
			assert.Equal(original.Headers, actual.Headers)
			assert.Equal(original.Metadata, actual.Metadata)
			assert.Equal(original.Spans, actual.Spans)
			assert.Equal(original.Path, actual.Path)
			assert.Equal(original.Objects, actual.Objects)
			assert.Equal(original.ServiceName, actual.ServiceName)
			assert.Equal(original.URL, actual.URL)
		}
	)

	{
		var (
			clone    = original
			response = clone.FailureResponse(nil, expectedSource, expectedRequestDeliveryResponse)
		)

		require.NotNil(response)
		checkResponse(response)
	}

	{
		var (
			clone            = original
			existingResponse Message
			response         = clone.FailureResponse(&existingResponse, expectedSource, expectedRequestDeliveryResponse)
		)

		require.NotNil(response)
		assert.Equal(&existingResponse, response)
		checkResponse(response)
	}

	{
		var (
			clone    = original
			response = clone.FailureResponse(&clone, expectedSource, expectedRequestDeliveryResponse)
		)

		require.NotNil(response)
		assert.Equal(&clone, response)
		checkResponse(response)
	}
}

func TestMessage(t *testing.T) {
	t.Run("SetStatus", testMessageSetStatus)
	t.Run("SetRequestDeliveryResponse", testMessageSetRequestDeliveryResponse)
	t.Run("SetIncludeSpans", testMessageSetIncludeSpans)

	var (
		expectedStatus                  int64 = 3471
		expectedRequestDeliveryResponse int64 = 34
		expectedIncludeSpans            bool  = true

		messages = []Message{
			Message{},
			Message{
				Type:   AuthMessageType,
				Status: &expectedStatus,
			},
			Message{
				Type:            SimpleEventMessageType,
				Source:          "mac:121234345656",
				Destination:     "foobar.com/service",
				TransactionUUID: "a unique identifier",
			},
			Message{
				Type:                    SimpleRequestResponseMessageType,
				Source:                  "somewhere.comcast.net:9090/something",
				Destination:             "serial:1234/blergh",
				TransactionUUID:         "123-123-123",
				Status:                  &expectedStatus,
				RequestDeliveryResponse: &expectedRequestDeliveryResponse,
				IncludeSpans:            &expectedIncludeSpans,
			},
			Message{
				Type:            SimpleRequestResponseMessageType,
				Source:          "external.com",
				Destination:     "mac:FFEEAADD44443333",
				TransactionUUID: "DEADBEEF",
				Headers:         []string{"Header1", "Header2"},
				Metadata:        map[string]string{"name": "value"},
				Spans:           [][]string{[]string{"1", "2"}, []string{"3"}},
				Payload:         []byte{1, 2, 3, 4, 0xff, 0xce},
			},
			Message{
				Type:        CreateMessageType,
				Source:      "wherever.webpa.comcast.net/glorious",
				Destination: "uuid:1111-11-111111-11111",
				Path:        "/some/where/over/the/rainbow",
				Objects:     "*",
			},
		}
	)

	t.Run("Routable", func(t *testing.T) {
		for _, message := range messages {
			testMessageRoutable(t, message)
		}
	})

	for _, source := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", source), func(t *testing.T) {
			for _, message := range messages {
				testMessageEncode(t, source, message)
			}
		})
	}

	t.Run("FailureResponse", func(t *testing.T) {
		for _, message := range messages {
			testMessageFailureResponse(t, message)
		}
	})
}

func testAuthorizationStatusEncode(t *testing.T, f Format) {
	var (
		assert   = assert.New(t)
		original = AuthorizationStatus{
			Status: 27,
		}

		decoded AuthorizationStatus

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.Equal(AuthMessageType, original.Type)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestAuthorizationStatus(t *testing.T) {
	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			testAuthorizationStatusEncode(t, format)
		})
	}
}

func testSimpleRequestResponseSetStatus(t *testing.T) {
	var (
		assert  = assert.New(t)
		message SimpleRequestResponse
	)

	assert.Nil(message.Status)
	assert.True(&message == message.SetStatus(15))
	assert.NotNil(message.Status)
	assert.Equal(int64(15), *message.Status)
	assert.True(&message == message.SetStatus(2312))
	assert.NotNil(message.Status)
	assert.Equal(int64(2312), *message.Status)
}

func testSimpleRequestResponseSetRequestDeliveryResponse(t *testing.T) {
	var (
		assert  = assert.New(t)
		message SimpleRequestResponse
	)

	assert.Nil(message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(2))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(2), *message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(67))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(67), *message.RequestDeliveryResponse)
}

func testSimpleRequestResponseSetIncludeSpans(t *testing.T) {
	var (
		assert  = assert.New(t)
		message SimpleRequestResponse
	)

	assert.Nil(message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(true))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(true, *message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(false))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(false, *message.IncludeSpans)
}

func testSimpleRequestResponseRoutable(t *testing.T, original SimpleRequestResponse) {
	assert := assert.New(t)
	assert.Equal(original.Type, original.MessageType())
	assert.Equal(original.Destination, original.To())
	assert.Equal(original.Source, original.From())
}

func testSimpleRequestResponseEncode(t *testing.T, f Format, original SimpleRequestResponse) {
	var (
		assert  = assert.New(t)
		decoded SimpleRequestResponse

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.Equal(SimpleRequestResponseMessageType, original.Type)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestSimpleRequestResponse(t *testing.T) {
	t.Run("SetStatus", testSimpleRequestResponseSetStatus)
	t.Run("SetRequestDeliveryResponse", testSimpleRequestResponseSetRequestDeliveryResponse)
	t.Run("SetIncludeSpans", testSimpleRequestResponseSetIncludeSpans)

	var (
		expectedStatus                  int64 = 121
		expectedRequestDeliveryResponse int64 = 17
		expectedIncludeSpans            bool  = true

		messages = []SimpleRequestResponse{
			SimpleRequestResponse{},
			SimpleRequestResponse{
				Source:          "mac:121234345656",
				Destination:     "foobar.com/service",
				TransactionUUID: "a unique identifier",
			},
			SimpleRequestResponse{
				Source:                  "somewhere.comcast.net:9090/something",
				Destination:             "serial:1234/blergh",
				TransactionUUID:         "123-123-123",
				Status:                  &expectedStatus,
				RequestDeliveryResponse: &expectedRequestDeliveryResponse,
				IncludeSpans:            &expectedIncludeSpans,
			},
			SimpleRequestResponse{
				Source:          "external.com",
				Destination:     "mac:FFEEAADD44443333",
				TransactionUUID: "DEADBEEF",
				Headers:         []string{"Header1", "Header2"},
				Metadata:        map[string]string{"name": "value"},
				Spans:           [][]string{[]string{"1", "2"}, []string{"3"}},
				Payload:         []byte{1, 2, 3, 4, 0xff, 0xce},
			},
		}
	)

	t.Run("Routable", func(t *testing.T) {
		for _, message := range messages {
			testSimpleRequestResponseRoutable(t, message)
		}
	})

	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			for _, message := range messages {
				testSimpleRequestResponseEncode(t, format, message)
			}
		})
	}
}

func testSimpleEventRoutable(t *testing.T, original SimpleEvent) {
	assert := assert.New(t)
	assert.Equal(original.Type, original.MessageType())
	assert.Equal(original.Destination, original.To())
	assert.Equal(original.Source, original.From())
}

func testSimpleEventEncode(t *testing.T, f Format, original SimpleEvent) {
	var (
		assert  = assert.New(t)
		decoded SimpleEvent

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.Equal(SimpleEventMessageType, original.Type)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestSimpleEvent(t *testing.T) {
	var messages = []SimpleEvent{
		SimpleEvent{},
		SimpleEvent{
			Source:      "simple.com/foo",
			Destination: "uuid:111111111111111",
			Payload:     []byte("this is a lovely payloed"),
		},
		SimpleEvent{
			Source:      "mac:123123123123123123",
			Destination: "something.webpa.comcast.net:9090/here/is/a/path",
			ContentType: "text/plain",
			Headers:     []string{"header1"},
			Metadata:    map[string]string{"a": "b", "c": "d"},
			Payload:     []byte("check this out!"),
		},
	}

	t.Run("Routable", func(t *testing.T) {
		for _, message := range messages {
			testSimpleEventRoutable(t, message)
		}
	})

	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			for _, message := range messages {
				testSimpleEventEncode(t, format, message)
			}
		})
	}
}

func testCRUDSetStatus(t *testing.T) {
	var (
		assert  = assert.New(t)
		message CRUD
	)

	assert.Nil(message.Status)
	assert.True(&message == message.SetStatus(-72))
	assert.NotNil(message.Status)
	assert.Equal(int64(-72), *message.Status)
	assert.True(&message == message.SetStatus(172))
	assert.NotNil(message.Status)
	assert.Equal(int64(172), *message.Status)
}

func testCRUDSetRequestDeliveryResponse(t *testing.T) {
	var (
		assert  = assert.New(t)
		message CRUD
	)

	assert.Nil(message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(123))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(123), *message.RequestDeliveryResponse)
	assert.True(&message == message.SetRequestDeliveryResponse(543543))
	assert.NotNil(message.RequestDeliveryResponse)
	assert.Equal(int64(543543), *message.RequestDeliveryResponse)
}

func testCRUDSetIncludeSpans(t *testing.T) {
	var (
		assert  = assert.New(t)
		message CRUD
	)

	assert.Nil(message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(true))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(true, *message.IncludeSpans)
	assert.True(&message == message.SetIncludeSpans(false))
	assert.NotNil(message.IncludeSpans)
	assert.Equal(false, *message.IncludeSpans)
}

func testCRUDRoutable(t *testing.T, original CRUD) {
	assert := assert.New(t)
	assert.Equal(original.Type, original.MessageType())
	assert.Equal(original.Destination, original.To())
	assert.Equal(original.Source, original.From())
}

func testCRUDEncode(t *testing.T, f Format, original CRUD) {
	var (
		assert  = assert.New(t)
		decoded CRUD

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestCRUD(t *testing.T) {
	t.Run("SetStatus", testCRUDSetStatus)
	t.Run("SetRequestDeliveryResponse", testCRUDSetRequestDeliveryResponse)
	t.Run("SetIncludeSpans", testCRUDSetIncludeSpans)

	var (
		expectedStatus                  int64 = -273
		expectedRequestDeliveryResponse int64 = 7223
		expectedIncludeSpans            bool  = true

		messages = []CRUD{
			CRUD{},
			CRUD{
				Type:            DeleteMessageType,
				Source:          "mac:121234345656",
				Destination:     "foobar.com/service",
				TransactionUUID: "a unique identifier",
				Path:            "/a/b/c/d",
			},
			CRUD{
				Type:                    CreateMessageType,
				Source:                  "somewhere.comcast.net:9090/something",
				Destination:             "serial:1234/blergh",
				TransactionUUID:         "123-123-123",
				Status:                  &expectedStatus,
				RequestDeliveryResponse: &expectedRequestDeliveryResponse,
				IncludeSpans:            &expectedIncludeSpans,
				Path:                    "/somewhere/over/rainbow",
				Objects:                 "asldkfja;sdkjfas;ldkjfasdkfj",
			},
			CRUD{
				Type:            UpdateMessageType,
				Source:          "external.com",
				Destination:     "mac:FFEEAADD44443333",
				TransactionUUID: "DEADBEEF",
				Headers:         []string{"Header1", "Header2"},
				Metadata:        map[string]string{"name": "value"},
				Spans:           [][]string{[]string{"1", "2"}, []string{"3"}},
			},
		}
	)

	t.Run("Routable", func(t *testing.T) {
		for _, message := range messages {
			testCRUDRoutable(t, message)
		}
	})

	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			for _, message := range messages {
				testCRUDEncode(t, format, message)
			}
		})
	}
}

func testServiceRegistrationEncode(t *testing.T, f Format, original ServiceRegistration) {
	var (
		assert  = assert.New(t)
		decoded ServiceRegistration

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.Equal(ServiceRegistrationMessageType, original.Type)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestServiceRegistration(t *testing.T) {
	var messages = []ServiceRegistration{
		ServiceRegistration{},
		ServiceRegistration{
			ServiceName: "systemd",
		},
		ServiceRegistration{
			ServiceName: "systemd",
			URL:         "local:/location/here",
		},
	}

	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			for _, message := range messages {
				testServiceRegistrationEncode(t, format, message)
			}
		})
	}
}

func testServiceAliveEncode(t *testing.T, f Format) {
	var (
		assert   = assert.New(t)
		original = ServiceAlive{}

		decoded ServiceAlive

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	assert.NoError(encoder.Encode(&original))
	assert.True(buffer.Len() > 0)
	assert.Equal(ServiceAliveMessageType, original.Type)
	assert.NoError(decoder.Decode(&decoded))
	assert.Equal(original, decoded)
}

func TestServiceAlive(t *testing.T) {
	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			testServiceAliveEncode(t, format)
		})
	}
}
