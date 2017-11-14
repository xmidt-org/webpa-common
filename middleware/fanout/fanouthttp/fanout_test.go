package fanouthttp

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDecodeFanoutRequestNilDecoder(t *testing.T, originalURL, relativeURL string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		original = httptest.NewRequest("GET", originalURL, bytes.NewReader([]byte("expected")))
		decoder  = decodeFanoutRequest(nil)
	)

	require.NotNil(decoder)
	v, err := decoder(context.Background(), original)
	require.NotNil(v)
	require.NoError(err)

	fanoutRequest := v.(*fanoutRequest)
	assert.True(original == fanoutRequest.original)
	assert.Equal(relativeURL, fanoutRequest.relativeURL.String())
	assert.Equal([]byte("expected"), fanoutRequest.entity)
}

func testDecodeFanoutRequestCustomDecoder(t *testing.T, originalURL, relativeURL string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCtx = context.WithValue(context.Background(), "foo", "bar")
		original    = httptest.NewRequest("GET", originalURL, bytes.NewReader([]byte("original body")))
		decoder     = decodeFanoutRequest(
			func(actualCtx context.Context, original *http.Request) (interface{}, error) {
				assert.Equal(expectedCtx, actualCtx)

				originalBody, err := ioutil.ReadAll(original.Body)
				assert.Equal([]byte("original body"), originalBody)
				assert.NoError(err)

				return "decoded body", nil
			},
		)
	)

	require.NotNil(decoder)
	v, err := decoder(expectedCtx, original)
	require.NotNil(v)
	require.NoError(err)

	fanoutRequest := v.(*fanoutRequest)
	assert.True(original == fanoutRequest.original)
	assert.Equal(relativeURL, fanoutRequest.relativeURL.String())
	assert.Equal("decoded body", fanoutRequest.entity)
}

func testDecodeFanoutRequestCustomDecoderError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedCtx   = context.WithValue(context.Background(), "foo", "bar")
		expectedError = errors.New("expected")
		original      = httptest.NewRequest("GET", "/does/not/matter", bytes.NewReader([]byte("original body")))
		decoder       = decodeFanoutRequest(
			func(actualCtx context.Context, original *http.Request) (interface{}, error) {
				assert.Equal(expectedCtx, actualCtx)

				originalBody, err := ioutil.ReadAll(original.Body)
				assert.Equal([]byte("original body"), originalBody)
				assert.NoError(err)

				return "decoded body", expectedError
			},
		)
	)

	require.NotNil(decoder)
	v, err := decoder(expectedCtx, original)
	assert.Nil(v)
	assert.Equal(expectedError, err)
}

func TestDecodeFanoutRequest(t *testing.T) {
	var testData = []struct {
		originalURL, relativeURL string
	}{
		{"http://localhost:8080", ""},
		{"http://localhost:8080/", "/"},
		{"https://something.comcast.net:2134/foo/bar", "/foo/bar"},
		{"https://something.comcast.net:2134/foo/bar?v=1&test=true", "/foo/bar?v=1&test=true"},
	}

	t.Run("NilDecoder", func(t *testing.T) {
		for _, record := range testData {
			testDecodeFanoutRequestNilDecoder(t, record.originalURL, record.relativeURL)
		}
	})

	t.Run("CustomDecoder", func(t *testing.T) {
		for _, record := range testData {
			testDecodeFanoutRequestCustomDecoder(t, record.originalURL, record.relativeURL)
		}
	})

	t.Run("CustomDecoderError", testDecodeFanoutRequestCustomDecoderError)
}

func testEncodeComponentRequestNilEncoder(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		original      = httptest.NewRequest("PATCH", "/foo/bar", nil)
		fanoutRequest = &fanoutRequest{
			original:    original,
			relativeURL: &url.URL{Path: "/foo/bar"},
			entity:      "decoded entity",
		}

		component = httptest.NewRequest("POST", "http://localhost:1234", nil)
		encoder   = encodeComponentRequest(nil)
	)

	require.NotNil(encoder)
	assert.NoError(encoder(context.Background(), component, fanoutRequest))
	assert.Equal(original.Method, component.Method)
	assert.Equal("http://localhost:1234/foo/bar", component.URL.String())
}

func testEncodeComponentRequestCustomEncoder(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		original      = httptest.NewRequest("PATCH", "/foo/bar", nil)
		fanoutRequest = &fanoutRequest{
			original:    original,
			relativeURL: &url.URL{Path: "/foo/bar"},
			entity:      "decoded entity",
		}

		expectedCtx         = context.WithValue(context.Background(), "foo", "bar")
		expectedComponent   = httptest.NewRequest("POST", "http://localhost:1234", nil)
		customEncoderCalled = false

		encoder = encodeComponentRequest(
			func(actualCtx context.Context, actualComponent *http.Request, v interface{}) error {
				assert.Equal(expectedCtx, actualCtx)
				assert.Equal(expectedComponent, actualComponent)
				assert.Equal("decoded entity", v)
				customEncoderCalled = true
				return nil
			},
		)
	)

	require.NotNil(encoder)
	assert.NoError(encoder(expectedCtx, expectedComponent, fanoutRequest))
	assert.Equal(original.Method, expectedComponent.Method)
	assert.Equal("http://localhost:1234/foo/bar", expectedComponent.URL.String())
	assert.True(customEncoderCalled)
}

func testEncodeComponentRequestCustomEncoderError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		original      = httptest.NewRequest("PATCH", "/foo/bar", nil)
		fanoutRequest = &fanoutRequest{
			original:    original,
			relativeURL: &url.URL{Path: "/foo/bar"},
			entity:      "decoded entity",
		}

		expectedCtx         = context.WithValue(context.Background(), "foo", "bar")
		expectedComponent   = httptest.NewRequest("POST", "http://localhost:1234", nil)
		expectedError       = errors.New("expected")
		customEncoderCalled = false

		encoder = encodeComponentRequest(
			func(actualCtx context.Context, actualComponent *http.Request, v interface{}) error {
				assert.Equal(expectedCtx, actualCtx)
				assert.Equal(expectedComponent, actualComponent)
				assert.Equal("decoded entity", v)
				customEncoderCalled = true
				return expectedError
			},
		)
	)

	require.NotNil(encoder)
	assert.Equal(expectedError, encoder(expectedCtx, expectedComponent, fanoutRequest))
	assert.Equal(original.Method, expectedComponent.Method)
	assert.Equal("http://localhost:1234/foo/bar", expectedComponent.URL.String())
	assert.True(customEncoderCalled)
}

func TestEncodeComponentRequest(t *testing.T) {
	t.Run("NilEncoder", testEncodeComponentRequestNilEncoder)
	t.Run("CustomEncoder", testEncodeComponentRequestCustomEncoder)
	t.Run("CustomEncoderError", testEncodeComponentRequestCustomEncoderError)
}

func testNewComponentsInvalidURL(t *testing.T) {
	assert := assert.New(t)
	for _, bad := range []string{"h\\ttp://localhost", "/foo/bar", "http://comcast.net:8080/test?v=1"} {
		components, err := NewComponents([]string{bad}, nil, nil)
		assert.Empty(components)
		assert.Error(err)
	}
}

func testNewComponentsSuccess(t *testing.T, urls ...string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		components, err = NewComponents(
			urls,
			func(context.Context, *http.Request, interface{}) error {
				return nil
			},
			func(context.Context, *http.Response) (interface{}, error) {
				return nil, nil
			},
		)
	)

	require.Equal(len(urls), len(components))
	require.NoError(err)

	for _, url := range urls {
		_, ok := components[url]
		assert.True(ok)
	}
}

func TestNewComponents(t *testing.T) {
	t.Run("InvalidURL", testNewComponentsInvalidURL)
	t.Run("Success", func(t *testing.T) {
		testNewComponentsSuccess(t, "http://something.comcast.net:8080")
		testNewComponentsSuccess(t, "http://somehost.com", "https://anotherhost.net:1212/foo/bar")
	})
}

func TestNewHandler(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		request  = httptest.NewRequest("GET", "http://localhost/foo/bar", nil)
		response = httptest.NewRecorder()

		handler = NewHandler(
			func(ctx context.Context, v interface{}) (interface{}, error) {
				fanoutRequest := v.(*fanoutRequest)
				assert.Equal(request, fanoutRequest.original)
				assert.Equal("/foo/bar", fanoutRequest.relativeURL.String())
				assert.Equal("decoded entity", fanoutRequest.entity)

				return "response", nil
			},
			func(ctx context.Context, original *http.Request) (interface{}, error) {
				return "decoded entity", nil
			},
			func(ctx context.Context, response http.ResponseWriter, v interface{}) error {
				assert.Equal("response", v)
				response.Header().Set("X-Processed", "true")
				return nil
			},
		)
	)

	require.NotNil(handler)
	handler.ServeHTTP(response, request)
	assert.Equal(response.Header().Get("X-Processed"), "true")
}
