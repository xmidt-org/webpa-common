package servicehttp

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/service"
)

func testKeyFromHeaderBlankHeader(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		KeyFromHeader("", func(string) (service.Key, error) { return nil, nil })
	})
}

func testKeyFromHeaderNilParser(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		KeyFromHeader("X-Something", nil)
	})
}

func testKeyFromHeaderMissingHeader(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	decoder := KeyFromHeader("X-Something", func(string) (service.Key, error) {
		assert.Fail("The parser should not have been called")
		return nil, nil
	})

	require.NotNil(decoder)

	key, err := decoder(context.Background(), httpRequest)
	assert.Error(err)
	assert.Nil(key)
}

func testKeyFromHeaderSuccess(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		httpRequest = httptest.NewRequest("GET", "/", nil)

		expectedHeaderValue = "expected header value"
		expectedKey         = service.StringKey("expected key")
	)

	decoder := KeyFromHeader("X-Something", func(v string) (service.Key, error) {
		assert.Equal(expectedHeaderValue, v)
		return expectedKey, nil
	})

	require.NotNil(decoder)
	httpRequest.Header.Set("X-Something", expectedHeaderValue)
	actualKey, err := decoder(context.Background(), httpRequest)
	assert.NoError(err)
	assert.Equal(expectedKey, actualKey)
}

func testKeyFromHeaderParserError(t *testing.T) {
	var (
		assert      = assert.New(t)
		require     = require.New(t)
		httpRequest = httptest.NewRequest("GET", "/", nil)

		expectedHeaderValue = "expected header value"
		expectedError       = errors.New("expected parser error")
	)

	decoder := KeyFromHeader("X-Something", func(v string) (service.Key, error) {
		assert.Equal(expectedHeaderValue, v)
		return nil, expectedError
	})

	require.NotNil(decoder)
	httpRequest.Header.Set("X-Something", expectedHeaderValue)
	key, actualError := decoder(context.Background(), httpRequest)
	assert.Equal(expectedError, actualError)
	assert.Nil(key)
}

func TestKeyFromHeader(t *testing.T) {
	t.Run("BlankHeader", testKeyFromHeaderBlankHeader)
	t.Run("NilParser", testKeyFromHeaderNilParser)
	t.Run("MissingHeader", testKeyFromHeaderMissingHeader)
	t.Run("Success", testKeyFromHeaderSuccess)
	t.Run("ParserError", testKeyFromHeaderParserError)
}

func testKeyFromPathBlankVariable(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		KeyFromPath("", func(string) (service.Key, error) { return nil, nil })
	})
}

func testKeyFromPathNilParser(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		KeyFromPath("id", nil)
	})
}

func testKeyFromPathSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedPathValue = "expected path value"
		expectedKey       = service.StringKey("expected key")
		httpRequest       = mux.SetURLVars(
			httptest.NewRequest("GET", "/", nil),
			map[string]string{"id": expectedPathValue},
		)
	)

	decoder := KeyFromPath("id", func(v string) (service.Key, error) {
		assert.Equal(expectedPathValue, v)
		return expectedKey, nil
	})

	require.NotNil(decoder)
	actualKey, err := decoder(context.Background(), httpRequest)
	assert.NoError(err)
	assert.Equal(expectedKey, actualKey)
}

func testKeyFromPathParserError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedPathValue = "expected path value"
		expectedError     = errors.New("expected parser error")
		httpRequest       = mux.SetURLVars(
			httptest.NewRequest("GET", "/", nil),
			map[string]string{"id": expectedPathValue},
		)
	)

	decoder := KeyFromPath("id", func(v string) (service.Key, error) {
		assert.Equal(expectedPathValue, v)
		return nil, expectedError
	})

	require.NotNil(decoder)
	key, actualError := decoder(context.Background(), httpRequest)
	assert.Equal(expectedError, actualError)
	assert.Nil(key)
}

func testKeyFromPathMissingVars(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	decoder := KeyFromPath("id", func(v string) (service.Key, error) {
		assert.Fail("The parser should not have been called")
		return nil, nil
	})

	require.NotNil(decoder)
	key, err := decoder(context.Background(), httpRequest)
	assert.Error(err)
	assert.Nil(key)
}

func testKeyFromPathMissingVariable(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		httpRequest = mux.SetURLVars(
			httptest.NewRequest("GET", "/", nil),
			map[string]string{"foo": "bar"},
		)
	)

	decoder := KeyFromPath("id", func(v string) (service.Key, error) {
		assert.Fail("The parser should not have been called")
		return nil, nil
	})

	require.NotNil(decoder)
	key, err := decoder(context.Background(), httpRequest)
	assert.Error(err)
	assert.Nil(key)
}

func TestKeyFromPath(t *testing.T) {
	t.Run("BlankHeader", testKeyFromPathBlankVariable)
	t.Run("NilParser", testKeyFromPathNilParser)
	t.Run("Success", testKeyFromPathSuccess)
	t.Run("ParserError", testKeyFromPathParserError)
	t.Run("MissingVars", testKeyFromPathMissingVars)
	t.Run("MissingVariable", testKeyFromPathMissingVariable)
}
