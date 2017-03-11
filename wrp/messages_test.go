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

func benchmarkEncodeDecode(b *testing.B, f Format, v interface{}) {
	var (
		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	b.ResetTimer()
	for repeat := 0; repeat < b.N; repeat++ {
		encoder.Encode(v)
		decoder.Decode(v)
	}
}

func BenchmarkRouting(b *testing.B) {
	for _, format := range allFormats {
		b.Run(format.String(), func(b *testing.B) {
			benchmarkEncodeDecode(b, format, new(Routing))
		})
	}
}

func BenchmarkMessage(b *testing.B) {
	for _, format := range allFormats {
		b.Run(format.String(), func(b *testing.B) {
			benchmarkEncodeDecode(b, format, new(Message))
		})
	}
}

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

func testAuthorizationStatusRouting(t *testing.T, f Format) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		original = AuthorizationStatus{
			Status: 13,
		}

		routing Routing

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	require.NoError(encoder.Encode(&original))
	assert.NoError(decoder.Decode(&routing))
	assert.Equal(AuthMessageType, routing.Type)
	assert.Empty(routing.Source)
	assert.Empty(routing.Destination)
}

func TestAuthorizationStatus(t *testing.T) {
	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			testAuthorizationStatusEncode(t, format)
		})

		t.Run(fmt.Sprintf("Routing%s", format), func(t *testing.T) {
			testAuthorizationStatusRouting(t, format)
		})
	}
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

func testMessageRouting(t *testing.T, f Format, original Message) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		routing Routing

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	require.NoError(encoder.Encode(&original))
	assert.NoError(decoder.Decode(&routing))
	assert.Equal(original.Source, routing.Source)
	assert.Equal(original.Destination, routing.Destination)
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

	for _, source := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", source), func(t *testing.T) {
			for _, message := range messages {
				testMessageEncode(t, source, message)
			}
		})

		t.Run(fmt.Sprintf("Routing%s", source), func(t *testing.T) {
			for _, message := range messages {
				testMessageRouting(t, source, message)
			}
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

func testSimpleRequestResponseRouting(t *testing.T, f Format, original SimpleRequestResponse) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		routing Routing

		buffer  bytes.Buffer
		encoder = NewEncoder(&buffer, f)
		decoder = NewDecoder(&buffer, f)
	)

	require.NoError(encoder.Encode(&original))
	assert.NoError(decoder.Decode(&routing))
	assert.Equal(SimpleRequestResponseMessageType, routing.Type)
	assert.Equal(original.Source, routing.Source)
	assert.Equal(original.Destination, routing.Destination)
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

	for _, format := range allFormats {
		t.Run(fmt.Sprintf("Encode%s", format), func(t *testing.T) {
			for _, message := range messages {
				testSimpleRequestResponseEncode(t, format, message)
			}
		})

		t.Run(fmt.Sprintf("Routing%s", format), func(t *testing.T) {
			for _, message := range messages {
				testSimpleRequestResponseRouting(t, format, message)
			}
		})
	}
}
