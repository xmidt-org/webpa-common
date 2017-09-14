package wrphttp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testClientEncodeRequestBodyEncodeError(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, wrp.JSON)

		wrpRequest = new(mockRequestResponse)
	)

	wrpRequest.On("Encode", mock.MatchedBy(func(io.Writer) bool { return true }), pool).Return(errors.New("expected")).Once()
	assert.Error(
		ClientEncodeRequestBody(pool)(context.Background(), httptest.NewRequest("GET", "/", nil), wrpRequest),
	)

	wrpRequest.AssertExpectations(t)
}

func testClientEncodeRequestBodySuccess(t *testing.T) {
	var (
		assert       = assert.New(t)
		pool         = wrp.NewEncoderPool(1, wrp.JSON)
		expectedBody = []byte("expected body")

		httpRequest = &http.Request{
			Header: http.Header{},
		}

		wrpRequest = new(mockRequestResponse)
	)

	wrpRequest.On("Encode", mock.MatchedBy(func(io.Writer) bool { return true }), pool).
		Run(func(arguments mock.Arguments) {
			output := arguments.Get(0).(io.Writer)
			output.Write(expectedBody)
		}).
		Return(error(nil)).Once()

	wrpRequest.On("Destination").Return("mac:101029293838").Once()

	assert.NoError(
		ClientEncodeRequestBody(pool)(context.Background(), httpRequest, wrpRequest),
	)

	assert.Equal("mac:101029293838", httpRequest.Header.Get(DestinationHeader))
	assert.Equal(pool.Format().ContentType(), httpRequest.Header.Get("Content-Type"))
	assert.Equal(int64(len(expectedBody)), httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Equal(expectedBody, actualBody)
	assert.NoError(err)

	wrpRequest.AssertExpectations(t)
}

func TestClientEncodeRequestBody(t *testing.T) {
	t.Run("EncodeError", testClientEncodeRequestBodyEncodeError)
	t.Run("Success", testClientEncodeRequestBodySuccess)
}

func testClientEncodeRequestHeadersNoPayload(t *testing.T) {
	var (
		assert = assert.New(t)

		message = &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "uuid:239487120398",
		}

		wrpRequest = new(mockRequestResponse)

		httpRequest = &http.Request{
			Header: http.Header{},
		}
	)

	wrpRequest.On("Message").Return(message).Twice()

	assert.NoError(
		ClientEncodeRequestHeaders(context.Background(), httpRequest, wrpRequest),
	)

	assert.Empty(httpRequest.Header.Get("Content-Type"))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpRequest.Header.Get(MessageTypeHeader))
	assert.Equal("test", httpRequest.Header.Get(SourceHeader))
	assert.Equal("uuid:239487120398", httpRequest.Header.Get(DestinationHeader))
	assert.Zero(httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Empty(actualBody)
	assert.NoError(err)

	wrpRequest.AssertExpectations(t)
}

func testClientEncodeRequestHeadersWithPayload(t *testing.T) {
	var (
		assert          = assert.New(t)
		expectedPayload = []byte("here is a lovely payload")

		message = &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "uuid:239487120398",
			ContentType: "text/plain",
			Payload:     expectedPayload,
		}

		wrpRequest = new(mockRequestResponse)

		httpRequest = &http.Request{
			Header: http.Header{},
		}
	)

	wrpRequest.On("Message").Return(message).Twice()

	assert.NoError(
		ClientEncodeRequestHeaders(context.Background(), httpRequest, wrpRequest),
	)

	assert.Equal("text/plain", httpRequest.Header.Get("Content-Type"))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpRequest.Header.Get(MessageTypeHeader))
	assert.Equal("test", httpRequest.Header.Get(SourceHeader))
	assert.Equal("uuid:239487120398", httpRequest.Header.Get(DestinationHeader))
	assert.Equal(int64(len(expectedPayload)), httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Equal(expectedPayload, actualBody)
	assert.NoError(err)

	wrpRequest.AssertExpectations(t)
}

func TestClientEncodeRequestHeaders(t *testing.T) {
	t.Run("NoPayload", testClientEncodeRequestHeadersNoPayload)
	t.Run("WithPayload", testClientEncodeRequestHeadersWithPayload)
}

func testServerEncodeResponseBodySuccess(t *testing.T, format wrp.Format) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, format)

		expectedPayload = []byte("expected payload")
		httpResponse    = httptest.NewRecorder()
		wrpResponse     = new(mockRequestResponse)
	)

	wrpResponse.On("Spans").Return([]tracing.Span{})
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
		wrpResponse  = new(mockRequestResponse)
	)

	wrpResponse.On("Spans").Return([]tracing.Span{})
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

		wrpResponse  = new(mockRequestResponse)
		httpResponse = httptest.NewRecorder()
	)

	wrpResponse.On("Spans").Return([]tracing.Span{})
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

		wrpResponse  = new(mockRequestResponse)
		httpResponse = httptest.NewRecorder()
	)

	wrpResponse.On("Spans").Return([]tracing.Span{})
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
