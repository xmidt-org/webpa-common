package wrpendpoint

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func assertLogger(t *testing.T, original Request, logger log.Logger) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	assert.NotNil(original.Logger())

	withNilLogger := original.WithLogger(nil)
	require.NotNil(withNilLogger)
	assert.NotNil(withNilLogger.Logger())
	assert.True(original != withNilLogger)
	assert.Equal(original.Message(), withNilLogger.Message())
	assertNote(t, *original.Message(), withNilLogger)

	newLogger := log.NewNopLogger()
	withLogger := original.WithLogger(newLogger)
	require.NotNil(withLogger)
	assert.NotNil(withLogger.Logger())
	assert.Equal(newLogger, withLogger.Logger())
	assert.True(original != withNilLogger)
	assert.Equal(original.Message(), withNilLogger.Message())
	assertNote(t, *original.Message(), withNilLogger)
}

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

func testDecodeRequest(t *testing.T, logger log.Logger, source io.Reader, format wrp.Format, original wrp.Message) {
	var (
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	request, err := DecodeRequest(logger, source, pool)
	require.NotNil(request)
	require.NoError(err)

	assertLogger(t, request, logger)
	assertNote(t, original, request)
}

func testDecodeRequestReadError(t *testing.T, format wrp.Format) {
	var (
		assert        = assert.New(t)
		pool          = wrp.NewDecoderPool(1, format)
		expectedError = errors.New("expected read error")
		source        = new(mockReader)
	)

	source.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	request, err := DecodeRequest(nil, source, pool)
	assert.Nil(request)
	assert.Equal(expectedError, err)
}

func testDecodeRequestBytes(t *testing.T, logger log.Logger, source []byte, format wrp.Format, original wrp.Message) {
	var (
		require = require.New(t)
		pool    = wrp.NewDecoderPool(1, format)
	)

	request, err := DecodeRequestBytes(logger, source, pool)
	require.NotNil(request)
	require.NoError(err)

	assertLogger(t, request, logger)
	assertNote(t, original, request)
}

func testDecodeRequestBytesDecodeError(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewDecoderPool(1, format)
	)

	request, err := DecodeRequestBytes(nil, []byte{0xFF}, pool)
	assert.Nil(request)
	assert.Error(err)
}

func testWrapAsRequest(t *testing.T, logger log.Logger, original wrp.Message) {
	request := WrapAsRequest(logger, &original)

	assertLogger(t, request, logger)
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
		t.Run(format.String(), func(t *testing.T) {
			t.Run("DecodeRequest", func(t *testing.T) {
				t.Run("NoLogger", func(t *testing.T) {
					testDecodeRequest(t, nil, bytes.NewReader(encoded), format, testMessage)
				})

				t.Run("WithLogger", func(t *testing.T) {
					testDecodeRequest(t, log.NewNopLogger(), bytes.NewReader(encoded), format, testMessage)
				})

				t.Run("ReadError", func(t *testing.T) {
					testDecodeRequestReadError(t, format)
				})
			})

			t.Run("DecodeRequestBytes", func(t *testing.T) {
				copyOf := make([]byte, len(encoded))
				copy(copyOf, encoded)

				t.Run("NoLogger", func(t *testing.T) {
					testDecodeRequestBytes(t, nil, copyOf, format, testMessage)
				})

				t.Run("WithLogger", func(t *testing.T) {
					testDecodeRequestBytes(t, log.NewNopLogger(), copyOf, format, testMessage)
				})

				t.Run("DecodeError", func(t *testing.T) {
					testDecodeRequestBytesDecodeError(t, format)
				})
			})

			t.Run("WrapAsRequest", func(t *testing.T) {
				t.Run("NoLogger", func(t *testing.T) {
					testWrapAsRequest(t, nil, testMessage)
				})

				t.Run("WithLogger", func(t *testing.T) {
					testWrapAsRequest(t, log.NewNopLogger(), testMessage)
				})
			})
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

func testDecodeResponseReadError(t *testing.T, format wrp.Format) {
	var (
		assert        = assert.New(t)
		pool          = wrp.NewDecoderPool(1, format)
		expectedError = errors.New("expected read error")
		source        = new(mockReader)
	)

	source.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	response, err := DecodeResponse(source, pool)
	assert.Nil(response)
	assert.Equal(expectedError, err)
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

func testDecodeResponseBytesDecodeError(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewDecoderPool(1, format)
	)

	response, err := DecodeResponseBytes([]byte{0xFF}, pool)
	assert.Nil(response)
	assert.Error(err)
}

func testWrapAsResponse(t *testing.T, original wrp.Message) {
	assertNote(t, original, WrapAsResponse(&original))
}

func testResponseSpans(t *testing.T, message wrp.Message) {
	var (
		require  = require.New(t)
		assert   = assert.New(t)
		spanner  = tracing.NewSpanner()
		original = WrapAsResponse(&message)
	)

	require.NotNil(original)
	assert.Empty(original.Spans())

	noNewSpans := original.AddSpans()
	require.NotNil(noNewSpans)
	assert.True(original == noNewSpans)
	assertNote(t, message, noNewSpans)

	newSpans := original.AddSpans(spanner.Start("first")(nil))
	require.NotNil(newSpans)
	assert.True(original != newSpans)
	assert.Equal(1, len(newSpans.Spans()))
	assert.Equal("first", newSpans.Spans()[0].Name())
	assert.NoError(newSpans.Spans()[0].Error())
	assertNote(t, message, newSpans)

	moreSpans := newSpans.AddSpans(spanner.Start("second")(errors.New("second error")), spanner.Start("third")(nil))
	require.NotNil(moreSpans)
	assert.True(newSpans != moreSpans)
	assert.Equal(3, len(moreSpans.Spans()))
	assert.Equal("first", moreSpans.Spans()[0].Name())
	assert.NoError(moreSpans.Spans()[0].Error())
	assert.Equal("second", moreSpans.Spans()[1].Name())
	assert.Error(errors.New("second error"), moreSpans.Spans()[1].Error())
	assert.Equal("third", moreSpans.Spans()[2].Name())
	assert.NoError(moreSpans.Spans()[2].Error())
	assertNote(t, message, moreSpans)
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
		t.Run(format.String(), func(t *testing.T) {
			t.Run("DecodeResponse", func(t *testing.T) {
				testDecodeResponse(t, bytes.NewReader(encoded), format, testMessage)

				t.Run("ReadError", func(t *testing.T) {
					testDecodeResponseReadError(t, format)
				})
			})

			t.Run("DecodeResponseBytes", func(t *testing.T) {
				copyOf := make([]byte, len(encoded))
				copy(copyOf, encoded)

				testDecodeResponseBytes(t, copyOf, format, testMessage)

				t.Run("DecodeError", func(t *testing.T) {
					testDecodeResponseBytesDecodeError(t, format)
				})
			})

			t.Run("Spans", func(t *testing.T) {
				testResponseSpans(t, testMessage)
			})
		})
	}
}
