package resource

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sort"
	"testing"
)

func TestData(t *testing.T) {
	assert := assert.New(t)

	message := "here are some resource bytes"
	var loader Loader = &Data{
		Source: []byte(message),
	}

	assert.Equal(message, loader.Location())
	assert.Equal(message, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(message, string(read))
	}

	// UseClient should have no effect
	UseClient(loader, &http.Client{})

	assert.Equal(message, loader.Location())
	assert.Equal(message, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(message, string(read))
	}
}

func TestFile(t *testing.T) {
	assert := assert.New(t)

	var loader Loader = &File{
		Path: testFilePath,
	}

	assert.Equal(testFilePath, loader.Location())
	assert.Equal(testFilePath, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(testContents, string(read))
	}

	// UseClient should have no effect
	UseClient(loader, &http.Client{})

	assert.Equal(testFilePath, loader.Location())
	assert.Equal(testFilePath, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(testContents, string(read))
	}
}

func TestFileMissing(t *testing.T) {
	assert := assert.New(t)
	missingFile := "/this/file/does/not/exist.txt"
	var loader Loader = &File{
		Path: missingFile,
	}

	assert.Equal(missingFile, loader.Location())
	assert.Equal(missingFile, fmt.Sprintf("%s", loader))

	read, err := ReadAll(loader)
	assert.Empty(read)
	assert.NotNil(err)

	// UseClient should have no effect
	UseClient(loader, &http.Client{})

	assert.Equal(missingFile, loader.Location())
	assert.Equal(missingFile, fmt.Sprintf("%s", loader))

	read, err = ReadAll(loader)
	assert.Empty(read)
	assert.NotNil(err)
}

func TestHTTPSimple(t *testing.T) {
	assert := assert.New(t)

	var loader Loader = &HTTP{
		URL: testFileURL,
	}

	assert.Equal(testFileURL, loader.Location())
	assert.Equal(testFileURL, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(testContents, string(read))
	}

	newClientUsed := false
	newClient := &testHTTPClient{
		transport: func(request *http.Request) (*http.Response, error) {
			newClientUsed = true
			return http.DefaultClient.Do(request)
		},
	}

	UseClient(loader, newClient)

	assert.Equal(testFileURL, loader.Location())
	assert.Equal(testFileURL, fmt.Sprintf("%s", loader))

	if read, err := ReadAll(loader); assert.Nil(err) {
		assert.Equal(testContents, string(read))
	}

	assert.True(newClientUsed)
}

func TestHTTPInvalidRequest(t *testing.T) {
	assert := assert.New(t)

	var loader Loader = &HTTP{
		URL:    testFileURL,
		Method: "INVALID METHOD",
	}

	data, err := ReadAll(loader)
	assert.Len(data, 0)
	assert.NotNil(err)
}

func TestHTTP(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		statusCode  int
		header      http.Header
		method      string
		clientError error
		expectError bool
	}{
		{
			statusCode: 200,
		},
		{
			statusCode: 204,
			header: http.Header{
				"Accept": []string{"text/plain"},
			},
		},
		{
			statusCode:  404,
			expectError: true,
		},
		{
			clientError: errors.New("here is a nasty little error!"),
			expectError: true,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		var body *readCloser

		client := &testHTTPClient{
			transport: func(request *http.Request) (*http.Response, error) {
				if len(record.method) == 0 {
					assert.Equal(DefaultMethod, request.Method)
				} else {
					assert.Equal(record.method, request.Method)
				}

				if assert.Len(request.Header, len(record.header)) {
					for key, actualValues := range request.Header {
						expectedValues := record.header[key]
						if assert.Len(actualValues, len(expectedValues)) && len(expectedValues) > 0 {
							sort.Strings(actualValues)
							sort.Strings(expectedValues)
							assert.Equal(expectedValues, actualValues)
						}
					}
				}

				if record.clientError != nil {
					return nil, record.clientError
				}

				body := &readCloser{
					reader: bytes.NewReader([]byte(testContents)),
				}

				return &http.Response{
					StatusCode: record.statusCode,
					Body:       body,
				}, nil
			},
		}

		var loader Loader = &HTTP{
			URL:        testFileURL,
			Header:     record.header,
			Method:     record.method,
			HTTPClient: client,
		}

		read, err := ReadAll(loader)
		if record.expectError {
			assert.Len(read, 0)
			assert.NotNil(err)
		} else {
			assert.Equal(testContents, string(read))
			assert.Nil(err)
		}

		assert.True(body == nil || body.closed)
	}
}
