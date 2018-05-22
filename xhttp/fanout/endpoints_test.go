package fanout

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	actualURLs, actualError := ef.FanoutURLs(original)
	assert.Equal(expectedURLs, actualURLs)
	assert.Equal(expectedError, actualError)
}

func testMustFanoutURLsPanics(t *testing.T) {
	var (
		assert    = assert.New(t)
		endpoints = new(mockEndpoints)
	)

	endpoints.On("FanoutURLs", mock.MatchedBy(func(*http.Request) bool { return true })).Return(nil, errors.New("expected")).Once()
	assert.Panics(func() {
		MustFanoutURLs(endpoints, httptest.NewRequest("GET", "/", nil))
	})

	endpoints.AssertExpectations(t)
}

func testMustFanoutURLsSuccess(t *testing.T) {
	var (
		assert       = assert.New(t)
		expectedURLs = []*url.URL{new(url.URL)}
		endpoints    = new(mockEndpoints)
	)

	endpoints.On("FanoutURLs", mock.MatchedBy(func(*http.Request) bool { return true })).Return(expectedURLs, error(nil)).Once()
	assert.NotPanics(func() {
		assert.Equal(expectedURLs, MustFanoutURLs(endpoints, httptest.NewRequest("GET", "/", nil)))
	})

	endpoints.AssertExpectations(t)
}

func TestMustFanoutURLs(t *testing.T) {
	t.Run("Panics", testMustFanoutURLsPanics)
	t.Run("Success", testMustFanoutURLsSuccess)
}

func testParseURLsEmpty(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = ParseURLs()
	)

	require.NotNil(fe)
	assert.Empty(fe)
	assert.NoError(err)
}

func testParseURLsInvalid(t *testing.T) {
	var (
		assert  = assert.New(t)
		fe, err = ParseURLs("%%")
	)

	assert.Empty(fe)
	assert.Error(err)
}

func testParseURLsValid(t *testing.T, urls []string, originalURL string, expected []string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = ParseURLs(urls...)
	)

	require.NotNil(fe)
	require.Len(fe, len(urls))
	require.NoError(err)

	actual, err := fe.FanoutURLs(httptest.NewRequest("GET", originalURL, nil))
	require.Equal(len(expected), len(actual))
	require.NoError(err)

	for i := 0; i < len(expected); i++ {
		assert.Equal(expected[i], actual[i].String())
	}
}

func TestParseURLs(t *testing.T) {
	t.Run("Empty", testParseURLsEmpty)
	t.Run("Invalid", testParseURLsInvalid)

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
			testParseURLsValid(t, record.urls, record.originalURL, record.expected)
		}
	})
}

func testMustParseURLsPanics(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		MustParseURLs("%%")
	})
}

func testMustParseURLsSuccess(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() {
		fe := MustParseURLs("http://foobar.com")
		assert.Len(fe, 1)
		assert.Equal("http://foobar.com", fe[0].String())
	})
}

func TestMustParseURLs(t *testing.T) {
	t.Run("Panics", testMustParseURLsPanics)
	t.Run("Success", testMustParseURLsSuccess)
}

func testNewEndpointsInvalidConfiguration(t *testing.T) {
	var (
		assert = assert.New(t)

		e, err = NewEndpoints(
			Configuration{Endpoints: []string{"%%"}},
			func() (Endpoints, error) {
				assert.Fail("The alternate function should not have been called")
				return nil, nil
			},
		)
	)

	assert.Nil(e)
	assert.Error(err)
}

func testNewEndpointsUseAlternate(t *testing.T) {
	var (
		assert = assert.New(t)

		expected    = MustParseURLs("http://localhost:1234")
		actual, err = NewEndpoints(
			Configuration{},
			func() (Endpoints, error) {
				return expected, nil
			},
		)
	)

	assert.Equal(expected, actual)
	assert.NoError(err)
}

func testNewEndpointsNoneConfigured(t *testing.T) {
	var (
		assert = assert.New(t)
		e, err = NewEndpoints(Configuration{}, nil)
	)

	assert.Nil(e)
	assert.Error(err)
}

func TestNewEndpoints(t *testing.T) {
	t.Run("InvalidConfiguration", testNewEndpointsInvalidConfiguration)
	t.Run("UseAlternate", testNewEndpointsUseAlternate)
	t.Run("NoneConfigured", testNewEndpointsNoneConfigured)
}

func testMustNewEndpointsPanics(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		MustNewEndpoints(Configuration{}, nil)
	})
}

func testMustNewEndpointsSuccess(t *testing.T) {
	var (
		assert   = assert.New(t)
		expected = MustParseURLs("http://foobar.com:1010")
	)

	assert.NotPanics(func() {
		assert.Equal(
			expected,
			MustNewEndpoints(Configuration{}, func() (Endpoints, error) { return expected, nil }),
		)
	})
}

func TestMustNewEndpoints(t *testing.T) {
	t.Run("Panics", testMustNewEndpointsPanics)
	t.Run("Success", testMustNewEndpointsSuccess)
}
