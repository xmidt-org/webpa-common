package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestParseHostPort(t *testing.T) {
	var (
		assert = assert.New(t)

		testData = []struct {
			value           string
			expectedBaseURL string
			expectsError    bool
		}{
			{"localhost:8080", "http://localhost:8080", false},
			{"[http://something.comcast.net]:8080", "http://something.comcast.net:8080", false},
			{"[https://65.71.145.16]:8080", "https://65.71.145.16:8080", false},
			{"", "", true},
			{"localhost", "", true},
			{"something.comcast.net", "", true},
			{"65.71.145.16", "", true},
		}
	)

	for _, record := range testData {
		t.Logf("%@v", record)

		actualBaseURL, err := ParseHostPort(record.value)
		assert.Equal(record.expectedBaseURL, actualBaseURL)
		assert.Equal(record.expectsError, err != nil)
	}
}

func TestReplaceHostPort(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			hostPort       string
			originalURL    url.URL
			expectedResult string
		}{
			{
				"http://localhost:8080",
				url.URL{},
				"http://localhost:8080",
			},
			{
				"https://something.comcast.net:16008",
				url.URL{
					Scheme: "http",
					Host:   "original.com",
					Path:   "/foo/bar",
				},
				"https://something.comcast.net:16008/foo/bar",
			},
			{
				"http://192.168.1.1:3412",
				url.URL{
					Scheme: "https",
					Host:   "original.com",
					Path:   "api/v2/device",
				},
				"http://192.168.1.1:3412/api/v2/device",
			},
			{
				"https://node1.comcast.net:342",
				url.URL{
					Scheme:     "https",
					Host:       "original.com",
					Path:       "api/v2/device",
					ForceQuery: true,
				},
				"https://node1.comcast.net:342/api/v2/device?",
			},
			{
				"http://28.77.145.1:9044",
				url.URL{
					Scheme:   "https",
					Host:     "another.original.com",
					Path:     "/list/something",
					RawQuery: "test=true&index=17",
				},
				"http://28.77.145.1:9044/list/something?test=true&index=17",
			},
			{
				"localhost:4455",
				url.URL{
					Scheme:   "https",
					Host:     "another.original.com",
					Path:     "path",
					RawQuery: "i=abc",
					Fragment: "location",
				},
				"localhost:4455/path?i=abc#location",
			},
			{
				"now.for.something.different.net",
				url.URL{
					Host:     "nopath.com",
					RawQuery: "i=abc",
					Fragment: "location",
				},
				"now.for.something.different.net?i=abc#location",
			},
			{
				"http://now.for.something.different.net",
				url.URL{
					Host:     "noquery.com",
					Fragment: "location",
				},
				"http://now.for.something.different.net#location",
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		actualResult := ReplaceHostPort(record.hostPort, &record.originalURL)
		assert.Equal(record.expectedResult, actualResult)
	}
}

func TestNewAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.TestLogger(t)

		testData = []struct {
			endpoints        []string
			expectedBaseURLs []string
			expectsError     bool
		}{
			{
				endpoints:        nil,
				expectedBaseURLs: make([]string, 0),
				expectsError:     true,
			},
			{
				endpoints:        []string{},
				expectedBaseURLs: make([]string, 0),
				expectsError:     true,
			},
			{
				endpoints:        []string{"[http://localhost]:7500"},
				expectedBaseURLs: []string{"http://localhost:7500"},
				expectsError:     false,
			},
			{
				endpoints:        []string{"[https://host1.net]:123", "[http://host2.com]:9293"},
				expectedBaseURLs: []string{"http://host2.com:9293", "https://host1.net:123"},
				expectsError:     false,
			},
			{
				endpoints:        []string{"localhost:8081", "this.is.not.valid", "[https://webpa.comcast.net]:7676"},
				expectedBaseURLs: []string{"http://localhost:8081", "https://webpa.comcast.net:7676"},
				expectsError:     false,
			},
			{
				endpoints:        []string{"this.is.not.valid"},
				expectedBaseURLs: []string{},
				expectsError:     true,
			},
			{
				endpoints:        []string{"this.is.not.valid", "neither.is.this"},
				expectedBaseURLs: []string{},
				expectsError:     true,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		for _, vnodeCount := range []uint{0, 200, 1700} {
			factory := NewAccessorFactory(&Options{Logger: logger, VnodeCount: vnodeCount})
			if !assert.NotNil(factory) {
				continue
			}

			accessor, actualBaseURLs := factory.New(record.endpoints)
			require.NotNil(accessor)
			assert.Equal(record.expectedBaseURLs, actualBaseURLs)

			endpoint, err := accessor.Get([]byte("key"))
			if record.expectsError {
				assert.Empty(endpoint)
				assert.Error(err)
			} else {
				assert.NotEmpty(endpoint)
				assert.NoError(err)
			}
		}
	}
}
