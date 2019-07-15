package logginghttp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestMethodKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(requestMethodKey, RequestMethodKey())
}

func TestRequestURIKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(requestURIKey, RequestURIKey())
}

func TestRemoteAddrKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(remoteAddrKey, RemoteAddrKey())
}

func TestRequestInfo(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = httptest.NewRequest("GET", "/test/foo/bar", nil)
	)

	request.RemoteAddr = "127.0.0.1:1234"

	kv := RequestInfo(nil, request)
	require.NotNil(kv)
	require.Len(kv, 6)
	assert.Equal(requestMethodKey, kv[0])
	assert.Equal("GET", kv[1])
	assert.Equal(requestURIKey, kv[2])
	assert.Equal("/test/foo/bar", kv[3])
	assert.Equal(remoteAddrKey, kv[4])
	assert.Equal("127.0.0.1:1234", kv[5])
}

func testHeaderMissing(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = httptest.NewRequest("GET", "/", nil)
	)

	kv := Header("X-Test", "key")(nil, request)
	require.NotNil(kv)
	require.Len(kv, 2)
	assert.Equal("key", kv[0])
	assert.Equal("", kv[1])
}

func testHeaderSingleValue(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = httptest.NewRequest("GET", "/", nil)
	)

	request.Header.Set("X-Test", "value")
	kv := Header("X-Test", "key")(nil, request)
	require.NotNil(kv)
	require.Len(kv, 2)
	assert.Equal("key", kv[0])
	assert.Equal("value", kv[1])
}

func testHeaderMultiValue(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = httptest.NewRequest("GET", "/", nil)
	)

	request.Header.Add("X-Test", "value1")
	request.Header.Add("X-Test", "value2")
	kv := Header("X-Test", "key")(nil, request)
	require.NotNil(kv)
	require.Len(kv, 2)
	assert.Equal("key", kv[0])
	assert.Equal([]string{"value1", "value2"}, kv[1])
}

func TestHeader(t *testing.T) {
	t.Run("Missing", testHeaderMissing)
	t.Run("SingleValue", testHeaderSingleValue)
	t.Run("MultiValue", testHeaderMultiValue)
}

func testPathVariableMissing(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = httptest.NewRequest("GET", "/", nil)
	)

	kv := PathVariable("test", "key")(nil, request)
	require.NotNil(kv)
	require.Len(kv, 2)
	assert.Equal("key", kv[0])
	assert.Equal("", kv[1])
}

func testPathVariableValue(t *testing.T) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		variables = map[string]string{
			"test": "foobar",
		}

		request = mux.SetURLVars(
			httptest.NewRequest("GET", "/", nil),
			variables,
		)
	)

	kv := PathVariable("test", "key")(nil, request)
	require.NotNil(kv)
	require.Len(kv, 2)
	assert.Equal("key", kv[0])
	assert.Equal("foobar", kv[1])
}

func TestPathVariable(t *testing.T) {
	t.Run("Missing", testPathVariableMissing)
	t.Run("Value", testPathVariableValue)
}

func testSetLoggerNilBase(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		SetLogger(nil)
	})
}

func testSetLoggerBaseOnly(t *testing.T) {
	var (
		assert = assert.New(t)

		base    = logging.NewTestLogger(nil, t)
		request = httptest.NewRequest("GET", "/", nil)
		ctx     = SetLogger(base)(context.Background(), request)
	)

	assert.Equal(base, logging.GetLogger(ctx))
}

func testSetLoggerCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		variables = map[string]string{
			"test": "path variable value",
		}

		request = mux.SetURLVars(
			httptest.NewRequest("GET", "/test/uri", nil),
			variables,
		)

		base = logging.NewCaptureLogger()
	)

	request.RemoteAddr = "10.0.0.1:7777"
	request.Header.Set("X-Test", "header value")

	logger := logging.GetLogger(
		SetLogger(
			base,
			RequestInfo, Header("X-Test", "key1"), PathVariable("test", "key2"),
		)(context.Background(), request),
	)

	require.NotEqual(base, logger)
	logger.Log(logging.MessageKey(), "test message")

	entry := <-base.Output()
	assert.Equal("GET", entry[requestMethodKey])
	assert.Equal("/test/uri", entry[requestURIKey])
	assert.Equal("10.0.0.1:7777", entry[remoteAddrKey])
	assert.Equal("header value", entry["key1"])
	assert.Equal("path variable value", entry["key2"])
	assert.Equal("test message", entry[logging.MessageKey()])
}

func TestSetLogger(t *testing.T) {
	t.Run("NilBase", testSetLoggerNilBase)
	t.Run("BaseOnly", testSetLoggerBaseOnly)
	t.Run("Custom", testSetLoggerCustom)
}
