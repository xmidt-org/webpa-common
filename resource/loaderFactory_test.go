package resource

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

type container struct {
	Resource LoaderFactory `json:"resource"`
}

func TestLoaderFactoryJSON(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		jsonValue     string
		loaderFactory LoaderFactory
	}{
		{
			`{"resource": "http://foobar.com"}`,
			LoaderFactory{
				location: "http://foobar.com",
			},
		},
		{
			`{"resource": "https://foobar.com"}`,
			LoaderFactory{
				location: "https://foobar.com",
			},
		},
		{
			`{"resource": "/etc/appname/config.txt"}`,
			LoaderFactory{
				location: "/etc/appname/config.txt",
			},
		},
		{
			`{"resource": {"buffer": "asdfasdfasdfasdfasdf"}}`,
			LoaderFactory{
				buffer: []byte("asdfasdfasdfasdfasdf"),
			},
		},
	}

	for _, record := range testData {
		{
			t.Logf("verifying Unmarshal(%v)", record.jsonValue)
			var container container
			if err := json.Unmarshal([]byte(record.jsonValue), &container); assert.Nil(err) {
				assert.Equal(record.loaderFactory, container.Resource)
			}
		}

		{
			t.Logf("verifying Marshal(%v)", record.loaderFactory)
			container := container{record.loaderFactory}
			if data, err := json.Marshal(container); assert.Nil(err) {
				assert.JSONEq(record.jsonValue, string(data))
			}
		}
	}
}

func TestLoaderFactoryNewLoader(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		loaderFactory LoaderFactory
		expected      Loader
	}{
		{
			LoaderFactory{
				location: "http://foobar.com",
			},
			&urlLoader{"http://foobar.com"},
		},
		{
			LoaderFactory{
				location: "https://foobar.com",
			},
			&urlLoader{"https://foobar.com"},
		},
		{
			LoaderFactory{
				location: "/etc/appname/config.txt",
			},
			&fileLoader{"/etc/appname/config.txt"},
		},
		{
			LoaderFactory{
				buffer: []byte("asdfasdfasdfasdfasdf"),
			},
			&bufferLoader{[]byte("asdfasdfasdfasdfasdf")},
		},
	}

	for _, record := range testData {
		assert.Equal(record.expected, record.loaderFactory.NewLoader())
	}
}
