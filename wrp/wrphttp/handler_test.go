package wrphttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedResponse ResponseWriter = &entityResponseWriter{}
		expectedRequest                 = new(Request)

		called             = false
		hf     HandlerFunc = func(actualResponse ResponseWriter, actualRequest *Request) {
			called = true
			assert.Equal(expectedResponse, actualResponse)
			assert.Equal(expectedRequest, actualRequest)
		}
	)

	hf.ServeWRP(expectedResponse, expectedRequest)
	assert.True(called)
}

func testWithErrorEncoderDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithErrorEncoder(nil)(wh)
	assert.NotNil(wh.errorEncoder)
}

func testWithErrorEncoderCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		customCalled                        = false
		custom       gokithttp.ErrorEncoder = func(context.Context, error, http.ResponseWriter) {
			customCalled = true
		}

		wh = new(wrpHandler)
	)

	WithErrorEncoder(custom)(wh)
	require.NotNil(wh.errorEncoder)

	wh.errorEncoder(context.Background(), errors.New("expected"), httptest.NewRecorder())
	assert.True(customCalled)
}

func TestWithErrorEncoder(t *testing.T) {
	t.Run("Default", testWithErrorEncoderDefault)
	t.Run("Custom", testWithErrorEncoderCustom)
}

func testWithNewResponseWriterDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		wh     = new(wrpHandler)
	)

	WithNewResponseWriter(nil)(wh)
	assert.NotNil(wh.newResponseWriter)
}

func testWithNewResponseWriterCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expected                    = &entityResponseWriter{}
		custom   ResponseWriterFunc = func(http.ResponseWriter, *Request) (ResponseWriter, error) {
			return expected, nil
		}

		wh = new(wrpHandler)
	)

	WithNewResponseWriter(custom)(wh)
	require.NotNil(wh.newResponseWriter)

	actual, err := wh.newResponseWriter(httptest.NewRecorder(), new(Request))
	assert.Equal(expected, actual)
	assert.NoError(err)
}

func TestWithNewResponseWriter(t *testing.T) {
	t.Run("Default", testWithNewResponseWriterDefault)
	t.Run("Custom", testWithNewResponseWriterCustom)
}
