package wrphttp

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testClientDecodeResponseBodyReadError(t *testing.T) {
	var (
		assert = assert.New(t)
		body   = new(mockReadCloser)
		pool   = wrp.NewDecoderPool(1, wrp.JSON)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
		}
	)

	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, errors.New("expected")).Once()
	value, err := ClientDecodeResponseBody(pool)(context.Background(), httpResponse)
	assert.Zero(pool.Len())
	assert.Nil(value)
	assert.Error(err)

	body.AssertExpectations(t)
}

func testClientDecodeResponseBodyHttpError(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewDecoderPool(1, wrp.JSON)

		httpResponse = &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader("dummy")),
		}
	)

	value, err := ClientDecodeResponseBody(pool)(context.Background(), httpResponse)
	assert.Zero(pool.Len())
	assert.Nil(value)
	assert.Error(err)
}

func testClientDecodeResponseBodyBadContentType(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewDecoderPool(1, wrp.JSON)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"bad content type"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`
				{"msg_type": 3, "source": "test", "dest": "mac:123443211234"}
			`)),
		}
	)

	value, err := ClientDecodeResponseBody(pool)(context.Background(), httpResponse)
	assert.Zero(pool.Len())
	assert.Nil(value)
	assert.Error(err)
}

func testClientDecodeResponseBodyUnexpectedContentType(t *testing.T) {
	var (
		assert = assert.New(t)
		pool   = wrp.NewDecoderPool(1, wrp.JSON)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/msgpack"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`
				{"msg_type": 3, "source": "test", "dest": "mac:123443211234"}
			`)),
		}
	)

	value, err := ClientDecodeResponseBody(pool)(context.Background(), httpResponse)
	assert.Zero(pool.Len())
	assert.Nil(value)
	assert.Error(err)
}

func testClientDecodeResponseBodySuccess(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
		pool    = wrp.NewDecoderPool(1, wrp.JSON)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: ioutil.NopCloser(strings.NewReader(`
				{"msg_type": 3, "source": "test", "dest": "mac:123443211234"}
			`)),
		}
	)

	value, err := ClientDecodeResponseBody(pool)(context.Background(), httpResponse)
	assert.Equal(1, pool.Len())
	require.NotNil(value)
	require.NoError(err)

	wrpResponse, ok := value.(wrpendpoint.Response)
	require.True(ok)

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleRequestResponseMessageType,
			Source:      "test",
			Destination: "mac:123443211234",
		},
		*wrpResponse.Message(),
	)
}

func TestClientDecodeResponseBody(t *testing.T) {
	t.Run("ReadError", testClientDecodeResponseBodyReadError)
	t.Run("HttpError", testClientDecodeResponseBodyHttpError)
	t.Run("BadContentType", testClientDecodeResponseBodyBadContentType)
	t.Run("UnexpectedContentType", testClientDecodeResponseBodyUnexpectedContentType)
	t.Run("Success", testClientDecodeResponseBodySuccess)
}

func testClientDecodeResponseHeadersReadError(t *testing.T) {
	var (
		assert = assert.New(t)
		body   = new(mockReadCloser)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
		}
	)

	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, errors.New("expected")).Once()
	value, err := ClientDecodeResponseHeaders(context.Background(), httpResponse)
	assert.Nil(value)
	assert.Error(err)

	body.AssertExpectations(t)
}

func testClientDecodeResponseHeadersHttpError(t *testing.T) {
	var (
		assert = assert.New(t)

		httpResponse = &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(strings.NewReader("dummy")),
		}
	)

	value, err := ClientDecodeResponseHeaders(context.Background(), httpResponse)
	assert.Nil(value)
	assert.Error(err)
}

func testClientDecodeResponseHeadersBadHeaders(t *testing.T) {
	var (
		assert = assert.New(t)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{}, // missing the message type
			Body:       ioutil.NopCloser(strings.NewReader("dummy")),
		}
	)

	value, err := ClientDecodeResponseHeaders(context.Background(), httpResponse)
	assert.Nil(value)
	assert.Error(err)
}

func testClientDecodeResponseHeadersNoPayload(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				MessageTypeHeader: []string{wrp.SimpleEventMessageType.FriendlyName()},
				SourceHeader:      []string{"test"},
				DestinationHeader: []string{"mac:019283745665"},
			},
			Body: ioutil.NopCloser(strings.NewReader("")),
		}
	)

	value, err := ClientDecodeResponseHeaders(context.Background(), httpResponse)
	require.NotNil(value)
	require.NoError(err)

	wrpResponse, ok := value.(wrpendpoint.Response)
	require.True(ok)

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:019283745665",
		},
		*wrpResponse.Message(),
	)
}

func testClientDecodeResponseHeadersWithPayload(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)

		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				MessageTypeHeader: []string{wrp.SimpleEventMessageType.FriendlyName()},
				SourceHeader:      []string{"test"},
				DestinationHeader: []string{"mac:019283745665"},
				"Content-Type":    []string{"text/plain"},
			},
			Body: ioutil.NopCloser(strings.NewReader("this is a payload")),
		}
	)

	value, err := ClientDecodeResponseHeaders(context.Background(), httpResponse)
	require.NotNil(value)
	require.NoError(err)

	wrpResponse, ok := value.(wrpendpoint.Response)
	require.True(ok)

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:019283745665",
			ContentType: "text/plain",
			Payload:     []byte("this is a payload"),
		},
		*wrpResponse.Message(),
	)
}

func TestClientDecodeResponseHeaders(t *testing.T) {
	t.Run("ReadError", testClientDecodeResponseHeadersReadError)
	t.Run("HttpError", testClientDecodeResponseHeadersHttpError)
	t.Run("BadHeaders", testClientDecodeResponseHeadersBadHeaders)
	t.Run("NoPayload", testClientDecodeResponseHeadersNoPayload)
	t.Run("WithPayload", testClientDecodeResponseHeadersWithPayload)
}

func TestServerDecodeRequestBody(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		pool        = wrp.NewDecoderPool(1, wrp.JSON)
		httpRequest = httptest.NewRequest("GET", "/", strings.NewReader(`
			{"msg_type": 3, "source": "test", "dest": "mac:123412341234"}
		`))
	)

	value, err := ServerDecodeRequestBody(logger, pool)(context.Background(), httpRequest)
	require.NotNil(value)
	require.NoError(err)

	wrpRequest, ok := value.(wrpendpoint.Request)
	require.True(ok)
	assert.NotNil(wrpRequest.Logger())

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleRequestResponseMessageType,
			Source:      "test",
			Destination: "mac:123412341234",
		},
		*wrpRequest.Message(),
	)
}

func testServerDecodeRequestHeadersSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.NewTestLogger(nil, t)

		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	httpRequest.Header.Set(MessageTypeHeader, "SimpleEvent")
	httpRequest.Header.Set(SourceHeader, "test")
	httpRequest.Header.Set(DestinationHeader, "mac:432143214321")

	value, err := ServerDecodeRequestHeaders(logger)(context.Background(), httpRequest)
	require.NotNil(value)
	require.NoError(err)

	wrpRequest, ok := value.(wrpendpoint.Request)
	require.True(ok)
	assert.NotNil(wrpRequest.Logger())

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:432143214321",
		},
		*wrpRequest.Message(),
	)
}

func testServerDecodeRequestHeadersBadHeaders(t *testing.T) {
	var (
		assert      = assert.New(t)
		logger      = logging.NewTestLogger(nil, t)
		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	value, err := ServerDecodeRequestHeaders(logger)(context.Background(), httpRequest)
	assert.Nil(value)
	assert.Error(err)
}

func TestServerDecodeRequestHeaders(t *testing.T) {
	t.Run("Success", testServerDecodeRequestHeadersSuccess)
	t.Run("BadHeaders", testServerDecodeRequestHeadersBadHeaders)
}
