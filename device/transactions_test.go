package device

import (
	"bytes"
	"context"
	"errors"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func testRequestContext(t *testing.T) {
	var (
		assert   = assert.New(t)
		message  = new(wrp.Message)
		format   = wrp.JSON
		contents = []byte("some contents")

		request = &Request{
			Message:  message,
			Format:   format,
			Contents: contents,
		}
	)

	assert.Equal(context.Background(), request.Context())
	assert.Panics(func() {
		request.WithContext(nil)
	})

	newContext := context.WithValue(context.Background(), "foo", "bar")
	assert.True(request == request.WithContext(newContext))
	assert.Equal(newContext, request.Context())
}

func testRequestID(t *testing.T) {
	var (
		assert  = assert.New(t)
		request = &Request{
			Message: &wrp.Message{
				Destination: "mac:123412341234",
			},
		}
	)

	id, err := request.ID()
	assert.Equal(ID("mac:123412341234"), id)
	assert.NoError(err)

	request.Message = &wrp.Message{
		Destination: "this is not a valid device ID",
	}

	id, err = request.ID()
	assert.Empty(string(id))
	assert.Error(err)
}

func TestRequest(t *testing.T) {
	t.Run("Context", testRequestContext)
	t.Run("ID", testRequestID)
}

func testDecodeRequest(t *testing.T, message wrp.Routable, format wrp.Format) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		contents []byte
		encoders = wrp.NewEncoderPool(1, format)
		decoders = wrp.NewDecoderPool(1, format)
	)

	require.NoError(encoders.EncodeBytes(&contents, message))

	request, err := DecodeRequest(bytes.NewReader(contents), decoders)
	require.NotNil(request)
	require.NoError(err)

	assert.Equal(message.MessageType(), request.Message.MessageType())
	assert.Equal(message.To(), request.Message.To())
	assert.Equal(message.From(), request.Message.From())
	assert.Equal(message.TransactionKey(), request.Message.TransactionKey())
	assert.Equal(format, request.Format)
	assert.Equal(contents, request.Contents)
	assert.Nil(request.ctx)
}

func testDecodeRequestReadError(t *testing.T, format wrp.Format) {
	var (
		assert        = assert.New(t)
		decoders      = wrp.NewDecoderPool(1, format)
		source        = new(mockReader)
		expectedError = errors.New("expected error")
	)

	source.On("Read", mock.AnythingOfType("[]uint8")).Return(0, expectedError)
	request, err := DecodeRequest(source, decoders)
	assert.Nil(request)
	assert.Equal(expectedError, err)

	source.AssertExpectations(t)
}

func testDecodeRequestDecodeError(t *testing.T, format wrp.Format) {
	var (
		assert   = assert.New(t)
		decoders = wrp.NewDecoderPool(1, format)
		empty    []byte
	)

	request, err := DecodeRequest(bytes.NewReader(empty), decoders)
	assert.Nil(request)
	assert.Error(err)
}

func TestDecodeRequest(t *testing.T) {
	for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
		t.Run(format.String(), func(t *testing.T) {
			testDecodeRequest(
				t,
				&wrp.SimpleEvent{
					Source:      "app.comcast.com:9999",
					Destination: "uuid:1234/service",
					ContentType: "text/plain",
					Payload:     []byte("hi there"),
				},
				format,
			)

			testDecodeRequest(
				t,
				&wrp.SimpleRequestResponse{
					Source:          "app.comcast.com:9999",
					Destination:     "uuid:1234/service",
					TransactionUUID: "this-is-a-transaction-id",
					ContentType:     "text/plain",
					Payload:         []byte("hi there"),
					Metadata:        map[string]string{"foo": "bar"},
				},
				format,
			)
		})
	}

	t.Run("ReadError", func(t *testing.T) {
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestReadError(t, format)
		}
	})

	t.Run("DecodeError", func(t *testing.T) {
		for _, format := range []wrp.Format{wrp.Msgpack, wrp.JSON} {
			testDecodeRequestDecodeError(t, format)
		}
	})
}
