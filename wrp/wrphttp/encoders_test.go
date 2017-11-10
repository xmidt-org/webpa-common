package wrphttp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testClientEncodeRequestBodyEncodeError(t *testing.T, custom http.Header) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewEncoderPool(1, wrp.JSON)

		wrpRequest = new(mockRequestResponse)
	)

	wrpRequest.On("Encode", mock.MatchedBy(func(io.Writer) bool { return true }), pool).Return(errors.New("expected")).Once()
	assert.Error(
		ClientEncodeRequestBody(pool, custom)(context.Background(), httptest.NewRequest("GET", "/", nil), wrpRequest),
	)

	wrpRequest.AssertExpectations(t)
}

func testClientEncodeRequestBodySuccess(t *testing.T, custom http.Header) {
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
		ClientEncodeRequestBody(pool, custom)(context.Background(), httpRequest, wrpRequest),
	)

	assert.Equal("mac:101029293838", httpRequest.Header.Get(DestinationHeader))
	assert.Equal(pool.Format().ContentType(), httpRequest.Header.Get("Content-Type"))
	assert.Equal(int64(len(expectedBody)), httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Equal(expectedBody, actualBody)
	assert.NoError(err)

	for name, value := range custom {
		assert.Equal(value, httpRequest.Header[name])
	}

	wrpRequest.AssertExpectations(t)
}

func TestClientEncodeRequestBody(t *testing.T) {
	t.Run("EncodeError", func(t *testing.T) {
		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestBodyEncodeError(t, nil)
		})

		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestBodyEncodeError(t,
				http.Header{"Accept": []string{"application/msgpack"}},
			)
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestBodySuccess(t, nil)
		})

		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestBodySuccess(t,
				http.Header{"Accept": []string{"application/msgpack"}},
			)
		})
	})
}

func testClientEncodeRequestHeadersNoPayload(t *testing.T, custom http.Header) {
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
		ClientEncodeRequestHeaders(custom)(context.Background(), httpRequest, wrpRequest),
	)

	assert.Empty(httpRequest.Header.Get("Content-Type"))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpRequest.Header.Get(MessageTypeHeader))
	assert.Equal("test", httpRequest.Header.Get(SourceHeader))
	assert.Equal("uuid:239487120398", httpRequest.Header.Get(DestinationHeader))
	assert.Zero(httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Empty(actualBody)
	assert.NoError(err)

	for name, value := range custom {
		assert.Equal(value, httpRequest.Header[name])
	}

	wrpRequest.AssertExpectations(t)
}

func testClientEncodeRequestHeadersWithPayload(t *testing.T, custom http.Header) {
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
		ClientEncodeRequestHeaders(custom)(context.Background(), httpRequest, wrpRequest),
	)

	assert.Equal("text/plain", httpRequest.Header.Get("Content-Type"))
	assert.Equal(wrp.SimpleEventMessageType.FriendlyName(), httpRequest.Header.Get(MessageTypeHeader))
	assert.Equal("test", httpRequest.Header.Get(SourceHeader))
	assert.Equal("uuid:239487120398", httpRequest.Header.Get(DestinationHeader))
	assert.Equal(int64(len(expectedPayload)), httpRequest.ContentLength)

	actualBody, err := ioutil.ReadAll(httpRequest.Body)
	assert.Equal(expectedPayload, actualBody)
	assert.NoError(err)

	for name, value := range custom {
		assert.Equal(value, httpRequest.Header[name])
	}

	wrpRequest.AssertExpectations(t)
}

func TestClientEncodeRequestHeaders(t *testing.T) {
	t.Run("NoPayload", func(t *testing.T) {
		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestHeadersNoPayload(t, nil)
		})

		t.Run("WithCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestHeadersNoPayload(t,
				http.Header{"Accept": []string{"application/msgpack"}},
			)
		})
	})

	t.Run("WithPayload", func(t *testing.T) {
		t.Run("NoCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestHeadersWithPayload(t, nil)
		})

		t.Run("WithCustomHeaders", func(t *testing.T) {
			testClientEncodeRequestHeadersWithPayload(t,
				http.Header{"Accept": []string{"application/msgpack"}},
			)
		})
	})
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

	assert.NoError(ServerEncodeResponseBody("", pool)(context.Background(), httpResponse, wrpResponse))
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

	assert.Error(ServerEncodeResponseBody("", pool)(context.Background(), httpResponse, wrpResponse))
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

	assert.NoError(ServerEncodeResponseHeaders("")(context.Background(), httpResponse, wrpResponse))
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

	assert.NoError(ServerEncodeResponseHeaders("")(context.Background(), httpResponse, wrpResponse))
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

func TestServerErrorEncoder(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			err                error
			expectedStatusCode int
			expectedHeader     http.Header
		}{
			{nil, http.StatusInternalServerError, http.Header{}},
			{errors.New("random error"), http.StatusInternalServerError, http.Header{}},
			{context.DeadlineExceeded, http.StatusGatewayTimeout, http.Header{}},
			{&httperror.E{Code: 403, Header: http.Header{"Foo": []string{"Bar"}}}, 403, http.Header{"Foo": []string{"Bar"}}},
			{tracing.NewSpanError(nil), http.StatusInternalServerError, http.Header{}},
			{tracing.NewSpanError(errors.New("random error")), http.StatusInternalServerError, http.Header{}},
			{tracing.NewSpanError(context.DeadlineExceeded), http.StatusGatewayTimeout, http.Header{}},
			{tracing.NewSpanError(&httperror.E{Code: 512, Header: http.Header{"Foo": []string{"Bar"}}}), http.StatusServiceUnavailable, http.Header{"Foo": []string{"Bar"}}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		response := httptest.NewRecorder()
		ServerErrorEncoder("")(context.Background(), record.err, response)
		assert.Equal(record.expectedStatusCode, response.Code)
		assert.Equal(record.expectedHeader, response.Header())
	}
}

func TestHeadersForError(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			err            error
			expectedHeader http.Header
		}{
			{nil, http.Header{}},
			{errors.New("random error"), http.Header{}},
			{context.DeadlineExceeded, http.Header{}},
			{&httperror.E{Header: http.Header{"Foo": []string{"Bar"}}}, http.Header{"Foo": []string{"Bar"}}},
			{tracing.NewSpanError(nil), http.Header{}},
			{tracing.NewSpanError(errors.New("random error")), http.Header{}},
			{tracing.NewSpanError(context.DeadlineExceeded), http.Header{}},
			{tracing.NewSpanError(&httperror.E{Header: http.Header{"Foo": []string{"Bar"}}}), http.Header{"Foo": []string{"Bar"}}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actualHeader := make(http.Header)
		HeadersForError(record.err, "", actualHeader)
		assert.Equal(record.expectedHeader, actualHeader)
	}
}

func TestStatusCodeForError(t *testing.T) {
	var (
		assert  = assert.New(t)
		spanner = tracing.NewSpanner()

		testData = []struct {
			err                error
			expectedStatusCode int
		}{
			{nil, http.StatusInternalServerError},
			{errors.New("random error"), http.StatusInternalServerError},
			{context.DeadlineExceeded, http.StatusGatewayTimeout},
			{&httperror.E{Code: 403}, 403},
			{tracing.NewSpanError(nil), http.StatusInternalServerError},
			{tracing.NewSpanError(errors.New("random error")), http.StatusInternalServerError},
			{tracing.NewSpanError(context.DeadlineExceeded), http.StatusGatewayTimeout},
			{tracing.NewSpanError(&httperror.E{Code: 403}), 403},
			{tracing.NewSpanError(nil), http.StatusInternalServerError},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusGatewayTimeout}),
				),
				http.StatusServiceUnavailable,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(context.DeadlineExceeded),
					spanner.Start("2")(&httperror.E{Code: http.StatusNotFound}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusNotFound}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("2")(&httperror.E{Code: http.StatusNotFound}),
					spanner.Start("3")(&httperror.E{Code: http.StatusInternalServerError}),
				),
				http.StatusNotFound,
			},

			{
				tracing.NewSpanError(errors.New("random error"),
					spanner.Start("1")(&httperror.E{Code: http.StatusInternalServerError}),
					spanner.Start("2")(&httperror.E{Code: http.StatusGatewayTimeout}),
					spanner.Start("3")(&httperror.E{Code: http.StatusNotFound}),
				),
				http.StatusNotFound,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		assert.Equal(record.expectedStatusCode, StatusCodeForError(record.err))
	}
}
