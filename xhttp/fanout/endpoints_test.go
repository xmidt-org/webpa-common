package fanout

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointsFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		original      = httptest.NewRequest("GET", "/", nil)
		expectedURLs  = []*url.URL{new(url.URL)}
		expectedError = errors.New("expected")

		ef = EndpointsFunc(func(actual *http.Request) ([]*url.URL, error) {
			assert.True(original == actual)
			return expectedURLs, expectedError
		})
	)

	actualURLs, actualError := ef.NewEndpoints(original)
	assert.Equal(expectedURLs, actualURLs)
	assert.Equal(expectedError, actualError)
}

func testNewFixedEndpointsEmpty(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = NewFixedEndpoints()
	)

	require.NotNil(fe)
	assert.Empty(fe)
	assert.NoError(err)
}

func testNewFixedEndpointsInvalid(t *testing.T) {
	var (
		assert  = assert.New(t)
		fe, err = NewFixedEndpoints("%%")
	)

	assert.Empty(fe)
	assert.Error(err)
}

func testNewFixedEndpointsValid(t *testing.T, urls []string, originalURL string, expected []string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = NewFixedEndpoints(urls...)
	)

	require.NotNil(fe)
	require.Len(fe, len(urls))
	require.NoError(err)

	actual, err := fe.NewEndpoints(httptest.NewRequest("GET", originalURL, nil))
	require.Equal(len(expected), len(actual))
	require.NoError(err)

	for i := 0; i < len(expected); i++ {
		assert.Equal(expected[i], actual[i].String())
	}
}

func TestNewFixedEndpoints(t *testing.T) {
	t.Run("Empty", testNewFixedEndpointsEmpty)
	t.Run("Invalid", testNewFixedEndpointsInvalid)

	t.Run("Valid", func(t *testing.T) {
		testData := []struct {
			urls        []string
			originalURL string
			expected    []string
		}{
			{
				[]string{"http://localhost:8080"},
				"/api/v2/something?value=1#mark",
				[]string{"http://localhost:8080/api/v2/something?value=1#mark"},
			},
			{
				[]string{"http://host1.someplace.com", "https://host2.someplace.net:1234"},
				"/api/v2/something",
				[]string{"http://host1.someplace.com/api/v2/something", "https://host2.someplace.net:1234/api/v2/something"},
			},
		}

		for _, record := range testData {
			testNewFixedEndpointsValid(t, record.urls, record.originalURL, record.expected)
		}
	})
}
