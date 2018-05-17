package xhttp

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testApplyURLParserEmpty(t *testing.T) {
	var (
		assert    = assert.New(t)
		urls, err = ApplyURLParser(url.Parse)
	)

	assert.Empty(urls)
	assert.NoError(err)
}

func testApplyURLParserError(t *testing.T, parser func(string) (*url.URL, error), values []string) {
	var (
		assert    = assert.New(t)
		urls, err = ApplyURLParser(parser, values...)
	)

	assert.Empty(urls)
	assert.Error(err)
}

func testApplyURLParserSuccess(t *testing.T, parser func(string) (*url.URL, error), values []string, expected []*url.URL) {
	var (
		assert      = assert.New(t)
		actual, err = ApplyURLParser(parser, values...)
	)

	assert.Equal(expected, actual)
	assert.NoError(err)
}

func TestApplyURLParser(t *testing.T) {
	t.Run("Empty", testApplyURLParserEmpty)

	t.Run("Error", func(t *testing.T) {
		testData := []struct {
			parser func(string) (*url.URL, error)
			values []string
		}{
			{url.Parse, []string{"%%"}},
			{url.Parse, []string{"http://localhost:1234", "%%"}},
			{url.Parse, []string{"%%", "https://foobar.net:8080"}},
			{url.Parse, []string{"http://something.net", "%%", "https://foobar.net:8080"}},
			{url.ParseRequestURI, []string{""}},
			{url.ParseRequestURI, []string{"http://localhost:1234", ""}},
			{url.ParseRequestURI, []string{"", "https://foobar.net:8080"}},
			{url.ParseRequestURI, []string{"http://something.net", "", "https://foobar.net:8080"}},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testApplyURLParserError(t, record.parser, record.values)
			})
		}
	})

	t.Run("Success", func(t *testing.T) {
		testData := []struct {
			parser   func(string) (*url.URL, error)
			values   []string
			expected []*url.URL
		}{
			{url.Parse, []string{"http://localhost"}, []*url.URL{&url.URL{Scheme: "http", Host: "localhost"}}},
			{url.Parse, []string{"https://foobar.net:8080", "http://localhost"}, []*url.URL{&url.URL{Scheme: "https", Host: "foobar.net:8080"}, &url.URL{Scheme: "http", Host: "localhost"}}},
			{url.ParseRequestURI, []string{"http://localhost"}, []*url.URL{&url.URL{Scheme: "http", Host: "localhost"}}},
			{url.ParseRequestURI, []string{"https://foobar.net:8080", "http://localhost"}, []*url.URL{&url.URL{Scheme: "https", Host: "foobar.net:8080"}, &url.URL{Scheme: "http", Host: "localhost"}}},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testApplyURLParserSuccess(t, record.parser, record.values, record.expected)
			})
		}
	})
}
