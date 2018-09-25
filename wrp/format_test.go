package wrp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPayload(t *testing.T, payload []byte) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		original = Message{
			Payload: payload,
		}

		decoded Message

		output  bytes.Buffer
		encoder = NewEncoder(nil, Msgpack)
		decoder = NewDecoder(nil, Msgpack)
	)

	encoder.Reset(&output)
	require.NoError(encoder.Encode(&original))

	decoder.Reset(&output)
	require.NoError(decoder.Decode(&decoded))

	// don't output the payload if it's a ridiculous size
	if testing.Verbose() && len(payload) < 1024 {
		fmt.Println(hex.Dump(output.Bytes()))
		t.Logf("original.Payload=%s", original.Payload)
		t.Logf("decoded.Payload=%s", decoded.Payload)
	}

	assert.Equal(payload, decoded.Payload)
}

func TestPayload(t *testing.T) {
	t.Run("UTF8", func(t *testing.T) {
		testPayload(t, []byte("this is clearly a UTF8 string"))
	})

	t.Run("Binary", func(t *testing.T) {
		testPayload(t, []byte{0x00, 0x06, 0xFF, 0xF0})
	})

	t.Run("LargePayload", func(t *testing.T) {
		// generate a very large random payload
		payload := make([]byte, 70*1024)
		rand.Read(payload)
		testPayload(t, payload)
	})
}

func TestSampleMsgpack(t *testing.T) {
	var (
		sampleEncoded = []byte{
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

		sampleMessage = SimpleRequestResponse{
			Type:            SimpleRequestResponseMessageType,
			Source:          "dns:webpa.comcast.com/v2-device-config",
			Destination:     "serial:1234/config",
			TransactionUUID: "9447241c-5238-4cb9-9baa-7076e3232899",
			Payload: []byte(
				`{ "names": [ "Device.X_CISCO_COM_Security.Firewall.FirewallLevel" ] }`,
			),
		}
	)

	t.Run("Encode", func(t *testing.T) {
		var (
			assert        = assert.New(t)
			buffer        bytes.Buffer
			encoder       = NewEncoder(&buffer, Msgpack)
			decoder       = NewDecoder(&buffer, Msgpack)
			actualMessage SimpleRequestResponse
		)

		assert.NoError(encoder.Encode(&sampleMessage))
		assert.NoError(decoder.Decode(&actualMessage))
		assert.Equal(sampleMessage, actualMessage)
	})

	t.Run("Decode", func(t *testing.T) {
		var (
			assert        = assert.New(t)
			decoder       = NewDecoder(bytes.NewBuffer(sampleEncoded), Msgpack)
			actualMessage SimpleRequestResponse
		)

		assert.NoError(decoder.Decode(&actualMessage))
		assert.Equal(sampleMessage, actualMessage)
	})

	t.Run("DecodeBytes", func(t *testing.T) {
		var (
			assert        = assert.New(t)
			decoder       = NewDecoderBytes(sampleEncoded, Msgpack)
			actualMessage SimpleRequestResponse
		)

		assert.NoError(decoder.Decode(&actualMessage))
		assert.Equal(sampleMessage, actualMessage)
	})
}

func testFormatFromContentTypeInvalid(t *testing.T, contentType string) {
	assert := assert.New(t)

	_, err := FormatFromContentType(contentType)
	assert.Error(err)

	// fallback won't matter if the content type is bad
	_, err = FormatFromContentType(contentType, Msgpack)
	assert.Error(err)

	_, err = FormatFromContentType(contentType, JSON)
	assert.Error(err)
}

func testFormatFromContentTypeValid(t *testing.T, contentType string, expected Format) {
	assert := assert.New(t)

	actual, err := FormatFromContentType(contentType)
	assert.Equal(expected, actual)
	assert.NoError(err)

	// For a valid content type, fallback won't matter
	actual, err = FormatFromContentType(contentType, Msgpack)
	assert.Equal(expected, actual)
	assert.NoError(err)

	actual, err = FormatFromContentType(contentType, JSON)
	assert.Equal(expected, actual)
	assert.NoError(err)
}

func testFormatFromContentTypeFallback(t *testing.T) {
	assert := assert.New(t)

	actual, err := FormatFromContentType("", Msgpack)
	assert.Equal(Msgpack, actual)
	assert.NoError(err)

	actual, err = FormatFromContentType("", JSON)
	assert.Equal(JSON, actual)
	assert.NoError(err)
}

func TestFormatFromContentType(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		testFormatFromContentTypeInvalid(t, "text/plain")
		testFormatFromContentTypeInvalid(t, "application/octet-stream")
	})

	t.Run("Valid", func(t *testing.T) {
		testFormatFromContentTypeValid(t, "application/msgpack", Msgpack)
		testFormatFromContentTypeValid(t, "application/json", JSON)
		testFormatFromContentTypeValid(t, "text/json", JSON)
	})

	t.Run("Fallback", testFormatFromContentTypeFallback)
}

func testFormatString(t *testing.T) {
	assert := assert.New(t)

	assert.NotEmpty(JSON.String())
	assert.NotEmpty(Msgpack.String())
	assert.NotEmpty(Format(-1).String())
	assert.NotEqual(JSON.String(), Msgpack.String())
}

func testFormatHandle(t *testing.T) {
	assert := assert.New(t)

	assert.NotNil(JSON.handle())
	assert.NotNil(Msgpack.handle())
	assert.Panics(func() { Format(999).handle() })
}

func testFormatContentType(t *testing.T) {
	assert := assert.New(t)

	assert.NotEmpty(JSON.ContentType())
	assert.NotEmpty(Msgpack.ContentType())
	assert.NotEqual(JSON.ContentType(), Msgpack.ContentType())
	assert.Equal("application/octet-stream", Format(999).ContentType())
}

func TestFormat(t *testing.T) {
	t.Run("String", testFormatString)
	t.Run("Handle", testFormatHandle)
	t.Run("ContentType", testFormatContentType)
}

// testTranscodeMessage expects a nonpointer reference to a WRP message struct as the original parameter
func testTranscodeMessage(t *testing.T, target, source Format, original interface{}) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		originalValue = reflect.ValueOf(original)
		encodeValue   = reflect.New(originalValue.Type())
		decodeValue   = reflect.New(originalValue.Type())
	)

	// encodeValue is now a pointer to a copy of the original
	encodeValue.Elem().Set(originalValue)

	var (
		sourceBuffer  bytes.Buffer
		sourceEncoder = NewEncoder(&sourceBuffer, source)
		sourceDecoder = NewDecoder(&sourceBuffer, source)

		targetBuffer  bytes.Buffer
		targetEncoder = NewEncoder(&targetBuffer, target)
		targetDecoder = NewDecoder(&targetBuffer, target)
	)

	// create the input first
	require.NoError(sourceEncoder.Encode(encodeValue.Interface()))

	// now we can attempt the transcode
	message, err := TranscodeMessage(targetEncoder, sourceDecoder)
	assert.NotNil(message)
	assert.NoError(err)

	assert.NoError(targetDecoder.Decode(decodeValue.Interface()))
	assert.Equal(encodeValue.Elem().Interface(), decodeValue.Elem().Interface())
}

func TestTranscodeMessage(t *testing.T) {
	var (
		expectedStatus                  int64 = 123
		expectedRequestDeliveryResponse int64 = -1234

		messages = []interface{}{
			SimpleRequestResponse{},
			SimpleRequestResponse{
				Source:      "foobar.com",
				Destination: "mac:FFEEDDCCBBAA",
				Payload:     []byte("hi!"),
			},
			SimpleRequestResponse{
				Source:                  "foobar.com",
				Destination:             "mac:FFEEDDCCBBAA",
				ContentType:             "application/wrp",
				Accept:                  "application/wrp",
				Status:                  &expectedStatus,
				RequestDeliveryResponse: &expectedRequestDeliveryResponse,
				Headers:                 []string{"X-Header-1", "X-Header-2"},
				Metadata:                map[string]string{"hi": "there"},
				Payload:                 []byte("hi!"),
			},
			Message{},
			Message{
				Source:      "foobar.com",
				Destination: "mac:FFEEDDCCBBAA",
				Payload:     []byte("hi!"),
			},
			Message{
				Source:                  "foobar.com",
				Destination:             "mac:FFEEDDCCBBAA",
				ContentType:             "application/wrp",
				Accept:                  "application/wrp",
				Status:                  &expectedStatus,
				RequestDeliveryResponse: &expectedRequestDeliveryResponse,
				Headers:                 []string{"X-Header-1", "X-Header-2"},
				Metadata:                map[string]string{"hi": "there"},
				Payload:                 []byte("hi!"),
			},
		}
	)

	for _, target := range allFormats {
		for _, source := range allFormats {
			t.Run(fmt.Sprintf("%sTo%s", source, target), func(t *testing.T) {
				for _, original := range messages {
					testTranscodeMessage(t, target, source, original)
				}
			})
		}
	}
}
