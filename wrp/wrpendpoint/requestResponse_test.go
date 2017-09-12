package wrpendpoint

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNoteEncodeUseContents(t *testing.T) {
	var (
		assert = assert.New(t)
		actual bytes.Buffer
		pool   = wrp.NewEncoderPool(1, wrp.Msgpack)

		note = note{
			contents: []byte("expected contents"),
			format:   wrp.Msgpack,
		}
	)

	assert.NoError(note.Encode(&actual, pool))
	assert.Equal("expected contents", actual.String())
	assert.Equal(0, pool.Len())
}

func testNoteEncodeUseMessage(t *testing.T) {
	var (
		assert = assert.New(t)
		actual bytes.Buffer
		pool   = wrp.NewEncoderPool(1, wrp.JSON)

		note = note{
			message: &wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Source:      "test",
				Destination: "test",
			},
		}
	)

	assert.NoError(note.Encode(&actual, pool))
	assert.JSONEq(`{"msg_type": 3, "source": "test", "dest": "test"}`, actual.String())
	assert.Equal(1, pool.Len())
}

func testNoteEncodeBytesUseContents(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, wrp.Msgpack)

		note = note{
			contents: []byte("expected contents"),
			format:   wrp.Msgpack,
		}
	)

	actual, err := note.EncodeBytes(pool)
	assert.Equal("expected contents", string(actual))
	assert.NoError(err)
	assert.Equal(0, pool.Len())
}

func testNoteEncodeBytesUseMessage(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, wrp.JSON)

		note = note{
			message: &wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Source:      "test",
				Destination: "test",
			},
		}
	)

	actual, err := note.EncodeBytes(pool)
	assert.NoError(err)
	assert.Equal(1, pool.Len())
	assert.JSONEq(`{"msg_type": 3, "source": "test", "dest": "test"}`, string(actual))
}

func TestNote(t *testing.T) {
	t.Run("Encode", func(t *testing.T) {
		t.Run("UseContents", testNoteEncodeUseContents)
		t.Run("UseMessage", testNoteEncodeUseMessage)
	})

	t.Run("EncodeBytes", func(t *testing.T) {
		t.Run("UseContents", testNoteEncodeBytesUseContents)
		t.Run("UseMessage", testNoteEncodeBytesUseMessage)
	})
}

func assertNote(t *testing.T, expected wrp.Message, actual Note) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	assert.Equal(expected.Destination, actual.Destination())
	assert.Equal(expected.TransactionUUID, actual.TransactionID())
	require.NotNil(actual.Message())
	assert.Equal(expected, *actual.Message())
}

func testDecodeRequest(t *testing.T, ctx context.Context, source io.Reader, format wrp.Format, original wrp.Message) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	request, err := DecodeRequest(ctx, source, pool)
	require.NotNil(request)
	require.NoError(err)

	if ctx != nil {
		assert.Equal(ctx, request.Context())
	} else {
		assert.Equal(context.Background(), request.Context())
	}

	assert.Panics(func() { request.WithContext(nil) })
	request2 := request.WithContext(context.WithValue(context.Background(), "test", true))
	require.NotNil(request2)
	assert.NotEqual(request2, request)
	assert.Equal(*request.Message(), *request2.Message())

	assertNote(t, original, request)
}

func testDecodeRequestBytes(t *testing.T, ctx context.Context, source []byte, format wrp.Format, original wrp.Message) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	request, err := DecodeRequestBytes(ctx, source, pool)
	require.NotNil(request)
	require.NoError(err)

	if ctx != nil {
		assert.Equal(ctx, request.Context())
	} else {
		assert.Equal(context.Background(), request.Context())
	}

	assert.Panics(func() { request.WithContext(nil) })
	request2 := request.WithContext(context.WithValue(context.Background(), "test", true))
	require.NotNil(request2)
	assert.NotEqual(request2, request)
	assert.Equal(*request.Message(), *request2.Message())

	assertNote(t, original, request)
}

func testWrapAsRequest(t *testing.T, ctx context.Context, original wrp.Message) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = WrapAsRequest(ctx, &original)
	)

	if ctx != nil {
		assert.Equal(ctx, request.Context())
	} else {
		assert.Equal(context.Background(), request.Context())
	}

	assert.Panics(func() { request.WithContext(nil) })
	request2 := request.WithContext(context.WithValue(context.Background(), "test", true))
	require.NotNil(request2)
	assert.NotEqual(request2, request)
	assert.Equal(*request.Message(), *request2.Message())

	assertNote(t, original, request)
}

func TestRequest(t *testing.T) {
	var (
		require     = require.New(t)
		testMessage = wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			TransactionUUID: "1234",
			Source:          "test",
			Destination:     "mac:111122223333",
			ContentType:     "text/plain",
			Payload:         []byte("hi!"),
		}
	)

	for _, format := range wrp.AllFormats() {
		var (
			encoded []byte
			encoder = wrp.NewEncoderBytes(&encoded, format)
		)

		require.NoError(encoder.Encode(&testMessage))

		t.Run("DecodeRequest", func(t *testing.T) {
			testDecodeRequest(t, nil, bytes.NewReader(encoded), format, testMessage)
			testDecodeRequest(t, context.WithValue(context.Background(), "foo", "bar"), bytes.NewReader(encoded), format, testMessage)
		})

		t.Run("DecodeRequestBytes", func(t *testing.T) {
			copyOf := make([]byte, len(encoded))
			copy(copyOf, encoded)

			testDecodeRequestBytes(t, nil, copyOf, format, testMessage)
			testDecodeRequestBytes(t, context.WithValue(context.Background(), "foo", "bar"), copyOf, format, testMessage)
		})

		t.Run("WrapAsRequest", func(t *testing.T) {
			testWrapAsRequest(t, nil, testMessage)
			testWrapAsRequest(t, context.WithValue(context.Background(), "foo", "bar"), testMessage)
		})
	}
}

func testDecodeResponse(t *testing.T, source io.Reader, format wrp.Format, original wrp.Message) {
	var (
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	response, err := DecodeResponse(source, pool)
	require.NotNil(response)
	require.NoError(err)

	assertNote(t, original, response)
}

func testDecodeResponseBytes(t *testing.T, source []byte, format wrp.Format, original wrp.Message) {
	var (
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	response, err := DecodeResponseBytes(source, pool)
	require.NotNil(response)
	require.NoError(err)

	assertNote(t, original, response)
}

func testWrapAsResponse(t *testing.T, original wrp.Message) {
	assertNote(t, original, WrapAsResponse(&original))
}

func TestResponse(t *testing.T) {
	var (
		require     = require.New(t)
		testMessage = wrp.Message{
			Type:            wrp.SimpleRequestResponseMessageType,
			TransactionUUID: "1234",
			Source:          "test",
			Destination:     "mac:111122223333",
			ContentType:     "text/plain",
			Payload:         []byte("hi!"),
		}
	)

	t.Run("WrapAsResponse", func(t *testing.T) {
		testWrapAsResponse(t, testMessage)
	})

	for _, format := range wrp.AllFormats() {
		var (
			encoded []byte
			encoder = wrp.NewEncoderBytes(&encoded, format)
		)

		require.NoError(encoder.Encode(&testMessage))

		t.Run("DecodeResponse", func(t *testing.T) {
			testDecodeResponse(t, bytes.NewReader(encoded), format, testMessage)
			testDecodeResponse(t, bytes.NewReader(encoded), format, testMessage)
		})

		t.Run("DecodeResponseBytes", func(t *testing.T) {
			copyOf := make([]byte, len(encoded))
			copy(copyOf, encoded)

			testDecodeResponseBytes(t, copyOf, format, testMessage)
			testDecodeResponseBytes(t, copyOf, format, testMessage)
		})
	}
}
