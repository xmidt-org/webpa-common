package fanouthttp

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
