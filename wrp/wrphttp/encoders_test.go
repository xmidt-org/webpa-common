package wrphttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testServerEncodeResponseBodySuccess(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, format)

		expectedPayload = []byte("expected payload")
		httpResponse    = httptest.NewRecorder()
		wrpResponse     = new(mockResponse)
	)

	wrpResponse.On("Encode", mock.MatchedBy(func(io.Writer) bool { return true }), pool).
		Run(func(arguments mock.Arguments) {
			output := arguments.Get(0).(io.Writer)
			output.Write(expectedPayload)
		}).
		Return(error(nil)).Once()

	assert.NoError(ServerEncodeResponseBody(pool)(context.Background(), httpResponse, wrpResponse))
	assert.Equal(http.StatusOK, httpResponse.Code)
	assert.Equal(format.ContentType(), httpResponse.HeaderMap.Get("Content-Type"))
	assert.Equal(expectedPayload, httpResponse.Body.Bytes())

	wrpResponse.AssertExpectations(t)
}

func testServerEncodeResponseBodyEncodeError(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, format)

		httpResponse = httptest.NewRecorder()
		wrpResponse  = new(mockResponse)
	)

	wrpResponse.On("Encode", mock.MatchedBy(func(io.Writer) bool { return true }), pool).
		Return(errors.New("expected error")).Once()

	assert.Error(ServerEncodeResponseBody(pool)(context.Background(), httpResponse, wrpResponse))
	assert.Empty(httpResponse.HeaderMap)
	assert.Empty(httpResponse.Body.Bytes())

	wrpResponse.AssertExpectations(t)
}

func TestServerEncodeResponseBody(t *testing.T) {
	for _, format := range wrp.AllFormats() {
		t.Run(format.String(), func(t *testing.T) {
			t.Run("Success", func(t *testing.T) {
				testServerEncodeResponseBodySuccess(t, format)
			})

			t.Run("EncodeError", func(t *testing.T) {
				testServerEncodeResponseBodyEncodeError(t, format)
			})
		})
	}
}

func testServerEncodeResponseHeadersNoPayload(t *testing.T) {
	var (
		assert = assert.New(t)

		message = wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:121212121212",
		}

		wrpResponse  = new(mockResponse)
		httpResponse = httptest.NewRecorder()
	)

	wrpResponse.On("Message").Return(&message).Twice()

	assert.NoError(ServerEncodeResponseHeaders(context.Background(), httpResponse, wrpResponse))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpResponse.HeaderMap.Get(MessageTypeHeader))
	assert.Equal("test", httpResponse.HeaderMap.Get(SourceHeader))
	assert.Equal("mac:121212121212", httpResponse.HeaderMap.Get(DestinationHeader))
	assert.Empty(httpResponse.HeaderMap.Get("Content-Type"))
	assert.Empty(httpResponse.Body.Bytes())

	wrpResponse.AssertExpectations(t)
}

func testServerEncodeResponseHeadersWithPayload(t *testing.T) {
	var (
		assert = assert.New(t)

		message = wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:121212121212",
			Payload:     []byte("expected payload"),
			ContentType: "text/plain",
		}

		wrpResponse  = new(mockResponse)
		httpResponse = httptest.NewRecorder()
	)

	wrpResponse.On("Message").Return(&message).Twice()

	assert.NoError(ServerEncodeResponseHeaders(context.Background(), httpResponse, wrpResponse))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpResponse.HeaderMap.Get(MessageTypeHeader))
	assert.Equal("test", httpResponse.HeaderMap.Get(SourceHeader))
	assert.Equal("mac:121212121212", httpResponse.HeaderMap.Get(DestinationHeader))
	assert.Equal("text/plain", httpResponse.HeaderMap.Get("Content-Type"))
	assert.Equal("expected payload", httpResponse.Body.String())

	wrpResponse.AssertExpectations(t)
}

func TestServerEncodeResponseHeaders(t *testing.T) {
	t.Run("NoPayload", testServerEncodeResponseHeadersNoPayload)
	t.Run("WithPayload", testServerEncodeResponseHeadersWithPayload)
}
