package wrp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func encodeBase64(input []byte) []byte {
	var output bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &output)
	_, err := encoder.Write(input)
	if err != nil {
		panic(err)
	}

	if err = encoder.Close(); err != nil {
		panic(err)
	}

	return output.Bytes()
}

func TestMessageTypeString(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		messageType    MessageType
		expectedString string
	}{
		{MessageType(0), InvalidMessageTypeString},
		{MessageType(1), InvalidMessageTypeString},
		{AuthMessageType, messageTypeStrings[AuthMessageType]},
		{SimpleRequestResponseMessageType, messageTypeStrings[SimpleRequestResponseMessageType]},
		{SimpleEventMessageType, messageTypeStrings[SimpleEventMessageType]},
		{MessageType(999), InvalidMessageTypeString},
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

func TestMessageSimpleRequestResponseDecode(t *testing.T) {
	assert := assert.New(t)

	input := bytes.NewReader(
		[]byte{0x85, 0xa8, 0x6d, 0x73, 0x67, 0x5f, 0x74, 0x79,
			0x70, 0x65, 0x03, 0xb0, 0x74, 0x72, 0x61, 0x6e,
			0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x75, 0x75,
			0x69, 0x64, 0xd9, 0x24, 0x39, 0x34, 0x34, 0x37,
			0x32, 0x34, 0x31, 0x63, 0x2d, 0x35, 0x32, 0x33,
			0x38, 0x2d, 0x34, 0x63, 0x62, 0x39, 0x2d, 0x39,
			0x62, 0x61, 0x61, 0x2d, 0x37, 0x30, 0x37, 0x36,
			0x65, 0x33, 0x32, 0x33, 0x32, 0x38, 0x39, 0x39,
			0xa6, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0xd9,
			0x26, 0x64, 0x6e, 0x73, 0x3a, 0x77, 0x65, 0x62,
			0x70, 0x61, 0x2e, 0x63, 0x6f, 0x6d, 0x63, 0x61,
			0x73, 0x74, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x76,
			0x32, 0x2d, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65,
			0x2d, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0xa4,
			0x64, 0x65, 0x73, 0x74, 0xb2, 0x73, 0x65, 0x72,
			0x69, 0x61, 0x6c, 0x3a, 0x31, 0x32, 0x33, 0x34,
			0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0xa7,
			0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0xc4,
			0x45, 0x7b, 0x20, 0x22, 0x6e, 0x61, 0x6d, 0x65,
			0x73, 0x22, 0x3a, 0x20, 0x5b, 0x20, 0x22, 0x44,
			0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x58, 0x5f,
			0x43, 0x49, 0x53, 0x43, 0x4f, 0x5f, 0x43, 0x4f,
			0x4d, 0x5f, 0x53, 0x65, 0x63, 0x75, 0x72, 0x69,
			0x74, 0x79, 0x2e, 0x46, 0x69, 0x72, 0x65, 0x77,
			0x61, 0x6c, 0x6c, 0x2e, 0x46, 0x69, 0x72, 0x65,
			0x77, 0x61, 0x6c, 0x6c, 0x4c, 0x65, 0x76, 0x65,
			0x6c, 0x22, 0x20, 0x5d, 0x20, 0x7d},
	)

	expectedPayload := []byte("{ \"names\": [ \"Device.X_CISCO_COM_Security.Firewall.FirewallLevel\" ] }")
	expectedPayloadBase64 := encodeBase64(expectedPayload)

	decoder := NewDecoder(input)
	var message Message
	assert.NotNil(message.Valid())

	err := decoder.Decode(&message)
	assert.Nil(err)
	assert.Equal(SimpleRequestResponseMessageType, message.Type)
	assert.Nil(message.Status)
	assert.Equal("dns:webpa.comcast.com/v2-device-config", message.Source)
	assert.Equal("serial:1234/config", message.Destination)
	assert.Equal("9447241c-5238-4cb9-9baa-7076e3232899", message.TransactionUUID)
	assert.Equal(expectedPayload, message.Payload)
	assert.Nil(message.Valid())

	rawJSON, err := json.Marshal(&message)
	assert.Nil(err)
	t.Logf("transformed json: %s", rawJSON)
	assert.JSONEq(
		fmt.Sprintf(`{
			"source": "dns:webpa.comcast.com/v2-device-config",
			"dest": "serial:1234/config",
			"transaction_uuid": "9447241c-5238-4cb9-9baa-7076e3232899",
			"payload": "%s"
		}`, expectedPayloadBase64),
		string(rawJSON),
	)
}

func TestMessageSerialization(t *testing.T) {
	assert := assert.New(t)
	expectedStatus := int64(123)

	expectedSource := "mac:112233445566"
	expectedDestination := "dns:somewhere.com/webhook"

	expectedPayload := "{ \"names\": [ \"Device.X_CISCO_COM_Security.Firewall.FirewallLevel\" ] }"
	expectedPayloadBase64 := string(
		encodeBase64(
			[]byte(expectedPayload),
		),
	)

	var testData = []struct {
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
				Payload:     []byte(expectedPayload),
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

	for _, record := range testData {
		t.Logf("%#v", record)

		var rawWRP bytes.Buffer
		wrpEncoder := NewEncoder(&rawWRP)
		err := wrpEncoder.Encode(&record.original)
		assert.Nil(err)
		assert.NotEmpty(rawWRP)

		var deserialized Message
		wrpDecoder := NewDecoder(&rawWRP)
		err = wrpDecoder.Decode(&deserialized)
		assert.Nil(err)
		assert.Equal(record.original, deserialized)

		rawJSON, err := json.Marshal(&record.original)
		assert.Nil(err)
		assert.JSONEq(record.expectedJSON, string(rawJSON))

		stringValue := record.original.String()
		assert.Contains(stringValue, record.original.Type.String())
		assert.Contains(stringValue, record.original.Source)
		assert.Contains(stringValue, record.original.Destination)
		assert.Contains(stringValue, fmt.Sprintf("%v", record.original.Payload))

		if record.original.Status != nil {
			assert.Contains(stringValue, strconv.FormatInt(*record.original.Status, 10))
		} else {
			assert.Contains(stringValue, "nil")
		}
	}
}

func TestMessageNilString(t *testing.T) {
	assert := assert.New(t)
	var message *Message
	assert.Equal("nil", message.String())
}
