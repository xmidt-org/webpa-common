package resource

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// Loader represents a type that can load data,
// potentially from outside the running process.
type Loader interface {
	// Location returns a string identifying where this Loader
	// gets its data from
	Location() string

	// Open returns a ReadCloser that reads this resource's data.
	Open() (io.ReadCloser, error)
}

// ReadAll is an analog to ioutil.ReadAll: it reads the entire
// resource into a single byte slice, returning any error that occurred.
func ReadAll(loader Loader) ([]byte, error) {
	reader, err := loader.Open()
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	return ioutil.ReadAll(reader)
}

// urlLoader implements Loader for a URL.
type urlLoader struct {
	url string
}

func (loader *urlLoader) Location() string {
	return loader.url
}

func (loader *urlLoader) Open() (reader io.ReadCloser, err error) {
	var response *http.Response
	response, err = http.Get(loader.url)
	defer func() {
		if err != nil && reader != nil {
			reader.Close()
			reader = nil
		}
	}()

	if response != nil {
		reader = response.Body
		if err == nil && response.StatusCode != 200 && response.StatusCode != 204 {
			err = fmt.Errorf(
				"Unable to access [%s]: server returned %s",
				loader.url,
				response.Status,
			)
		}
	}

	return
}

type fileLoader struct {
	file string
}

func (loader *fileLoader) Location() string {
	return loader.file
}

func (loader *fileLoader) Open() (reader io.ReadCloser, err error) {
	reader, err = os.Open(loader.file)
	if err != nil && reader != nil {
		reader.Close()
		reader = nil
	}

	return
}

type readCloserAdapter struct {
	io.Reader
}

func (a readCloserAdapter) Close() error {
	return nil
}

type bufferLoader struct {
	source []byte
}

func (loader *bufferLoader) Location() string {
	return "buffer"
}

func (loader *bufferLoader) Open() (io.ReadCloser, error) {
	return readCloserAdapter{bytes.NewReader(loader.source)}, nil
}
