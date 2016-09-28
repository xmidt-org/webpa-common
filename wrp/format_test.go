package wrp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func encodeBase64(input string) []byte {
	var output bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &output)
	_, err := encoder.Write([]byte(input))
	if err != nil {
		panic(err)
	}

	if err = encoder.Close(); err != nil {
		panic(err)
	}

	return output.Bytes()
}

var (
	// simpleRequestResponseMsgpack is a hand-coded example of a valid request/response message
	// in msgpack format
	simpleRequestResponseMsgpack = []byte{
		0x85, 0xa8, 0x6d, 0x73, 0x67, 0x5f, 0x74, 0x79,
		0x70, 0x65, 0x03, 0xb0, 0x74, 0x72, 0x61, 0x6e,
		0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
		0x75, 0x75, 0x69, 0x64, 0xd9, 0x24, 0x39, 0x34,
		0x34, 0x37, 0x32, 0x34, 0x31, 0x63, 0x2d, 0x35,
		0x32, 0x33, 0x38, 0x2d, 0x34, 0x63, 0x62, 0x39,
		0x2d, 0x39, 0x62, 0x61, 0x61, 0x2d, 0x37, 0x30,
		0x37, 0x36, 0x65, 0x33, 0x32, 0x33, 0x32, 0x38,
		0x39, 0x39, 0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63,
		0x65, 0xd9, 0x26, 0x64, 0x6e, 0x73, 0x3a, 0x77,
		0x65, 0x62, 0x70, 0x61, 0x2e, 0x63, 0x6f, 0x6d,
		0x63, 0x61, 0x73, 0x74, 0x2e, 0x63, 0x6f, 0x6d,
		0x2f, 0x76, 0x32, 0x2d, 0x64, 0x65, 0x76, 0x69,
		0x63, 0x65, 0x2d, 0x63, 0x6f, 0x6e, 0x66, 0x69,
		0x67, 0xa4, 0x64, 0x65, 0x73, 0x74, 0xb2, 0x73,
		0x65, 0x72, 0x69, 0x61, 0x6c, 0x3a, 0x31, 0x32,
		0x33, 0x34, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
		0x67, 0xa7, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61,
		0x64, 0xc4, 0x45, 0x7b, 0x20, 0x22, 0x6e, 0x61,
		0x6d, 0x65, 0x73, 0x22, 0x3a, 0x20, 0x5b, 0x20,
		0x22, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e,
		0x58, 0x5f, 0x43, 0x49, 0x53, 0x43, 0x4f, 0x5f,
		0x43, 0x4f, 0x4d, 0x5f, 0x53, 0x65, 0x63, 0x75,
		0x72, 0x69, 0x74, 0x79, 0x2e, 0x46, 0x69, 0x72,
		0x65, 0x77, 0x61, 0x6c, 0x6c, 0x2e, 0x46, 0x69,
		0x72, 0x65, 0x77, 0x61, 0x6c, 0x6c, 0x4c, 0x65,
		0x76, 0x65, 0x6c, 0x22, 0x20, 0x5d, 0x20, 0x7d,
	}

	// expectedPayload is the payload within simpleRequestResponseMsgpack
	expectedPayload       = `{ "names": [ "Device.X_CISCO_COM_Security.Firewall.FirewallLevel" ] }`
	expectedPayloadBase64 = encodeBase64(expectedPayload)

	expectedStatus      = int64(123)
	expectedSource      = "mac:112233445566"
	expectedDestination = "dns:somewhere.com/webhook"

	encoderTestData = []struct {
		original     Message
		expectedJSON string
	}{
		{
			original: Message{
				Type:   AuthMessageType,
				Status: &expectedStatus,
			},
			expectedJSON: `
				{"status": 123}
			`,
		},
		{
			original: Message{
				Type:        SimpleEventMessageType,
				Destination: expectedDestination,
				Payload:     expectedPayloadBase64,
			},
			expectedJSON: fmt.Sprintf(`{
				"dest": "%s",
				"payload": "%s"
			}`, expectedDestination, expectedPayloadBase64),
		},
		{
			original: Message{
				Type:        SimpleRequestResponseMessageType,
				Source:      expectedSource,
				Destination: expectedDestination,
				Payload:     []byte(expectedPayload),
			},
			expectedJSON: fmt.Sprintf(`{
				"source": "%s",
				"dest": "%s",
				"payload": "%s"
			}`, expectedSource, expectedDestination, expectedPayloadBase64),
		},
	}
)

// assertSimpleRequestResponse is an assertion specific to the hand-coded request/response message
func assertSimpleRequestResponse(assert *assert.Assertions, actual *Message) {
	assert.Equal(SimpleRequestResponseMessageType, actual.Type)
	assert.Nil(actual.Status)
	assert.Equal("dns:webpa.comcast.com/v2-device-config", actual.Source)
	assert.Equal("serial:1234/config", actual.Destination)
	assert.Equal("9447241c-5238-4cb9-9baa-7076e3232899", actual.TransactionUUID)
	assert.Equal([]byte(expectedPayload), actual.Payload)
	assert.Nil(actual.Valid())
}

// assertStringValue runs some sanity checks on the String() representation of a message.
// This is important because we want certain items output in logs.
func assertStringValue(assert *assert.Assertions, actual *Message) {
	stringValue := actual.String()
	assert.Contains(stringValue, actual.Type.String())
	assert.Contains(stringValue, actual.Source)
	assert.Contains(stringValue, actual.Destination)
	assert.Contains(stringValue, fmt.Sprintf("%v", actual.Payload))

	if actual.Status != nil {
		assert.Contains(stringValue, strconv.FormatInt(*actual.Status, 10))
	} else {
		assert.Contains(stringValue, "nil")
	}
}

func TestDecoderBytesMsgpackSimpleRequestResponse(t *testing.T) {
	assert := assert.New(t)

	var message Message
	assert.NotNil(message.Valid())

	decoder := NewDecoderBytes(simpleRequestResponseMsgpack, Msgpack)
	err := decoder.Decode(&message)
	assert.Nil(err)
	assertSimpleRequestResponse(assert, &message)
}

func TestDecoderMsgpackSimpleRequestResponse(t *testing.T) {
	assert := assert.New(t)

	var message Message
	assert.NotNil(message.Valid())

	output := bytes.NewBuffer(simpleRequestResponseMsgpack)
	decoder := NewDecoder(output, Msgpack)
	err := decoder.Decode(&message)
	assert.Nil(err)
	t.Logf("%s", &message)
	assertSimpleRequestResponse(assert, &message)
}

func TestEncoderMsgpack(t *testing.T) {
	assert := assert.New(t)

	for _, record := range encoderTestData {
		t.Logf("%#v", record)

		var serialized bytes.Buffer
		encoder := NewEncoder(&serialized, Msgpack)
		assert.Nil(encoder.Encode(&record.original))
		assert.NotEmpty(serialized)

		var deserialized Message
		decoder := NewDecoder(&serialized, Msgpack)
		assert.Nil(decoder.Decode(&deserialized))
		assert.Equal(record.original, deserialized)

		assertStringValue(assert, &deserialized)
	}
}

func TestEncoderBytesMsgpack(t *testing.T) {
	assert := assert.New(t)

	for _, record := range encoderTestData {
		t.Logf("%#v", record)

		var serialized []byte
		encoder := NewEncoderBytes(&serialized, Msgpack)
		assert.Nil(encoder.Encode(&record.original))
		assert.NotEmpty(serialized)

		var deserialized Message
		decoder := NewDecoderBytes(serialized, Msgpack)
		assert.Nil(decoder.Decode(&deserialized))
		assert.Equal(record.original, deserialized)

		assertStringValue(assert, &deserialized)
	}
}

func TestEncoderJSON(t *testing.T) {
	assert := assert.New(t)

	for _, record := range encoderTestData {
		t.Logf("%#v", record)

		var serialized bytes.Buffer
		encoder := NewEncoder(&serialized, JSON)
		assert.Nil(encoder.Encode(&record.original))
		assert.NotEmpty(serialized)

		var deserialized Message
		decoder := NewDecoder(&serialized, JSON)
		assert.Nil(decoder.Decode(&deserialized))
		assert.Equal(MessageType(0), deserialized.Type)
		assert.Nil(deserialized.DeduceType())
		assert.Equal(record.original, deserialized)

		assertStringValue(assert, &deserialized)
	}
}

func TestEncoderBytesJSON(t *testing.T) {
	assert := assert.New(t)

	for _, record := range encoderTestData {
		t.Logf("%#v", record)

		var serialized []byte
		encoder := NewEncoderBytes(&serialized, JSON)
		assert.Nil(encoder.Encode(&record.original))
		assert.NotEmpty(serialized)

		var deserialized Message
		decoder := NewDecoderBytes(serialized, JSON)
		assert.Nil(decoder.Decode(&deserialized))
		assert.Equal(MessageType(0), deserialized.Type)
		assert.Nil(deserialized.DeduceType())
		assert.Equal(record.original, deserialized)

		assertStringValue(assert, &deserialized)
	}
}

func TestFormatHandle(t *testing.T) {
	assert := assert.New(t)

	assert.NotNil(JSON.handle())
	assert.NotNil(Msgpack.handle())
	assert.Nil(Format(999).handle())
}
