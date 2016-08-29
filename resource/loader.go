package resource

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	// DefaultMethod is the default HTTP method used when none is supplied
	DefaultMethod = "GET"
)

// httpClient is an internal strategy interface for objects which
// can handle HTTP transactions.  *http.Client implements this interface.
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Loader represents a type that can load data,
// potentially from outside the running process.
type Loader interface {
	// Location returns a string identifying where this Loader
	// gets its data from
	Location() string

	// Open returns a ReadCloser that reads this resource's data.
	Open() (io.ReadCloser, error)
}

// UseClient will change the HTTP client object used by the given resource.  If loader
// is not an HTTP Loader, this function does nothing.  A nil client will cause the
// loader to use http.DefaultClient.
func UseClient(loader Loader, HTTPClient httpClient) {
	if httpLoader, ok := loader.(*HTTP); ok {
		httpLoader.HTTPClient = HTTPClient
	}
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

// HTTP is a Loader which obtains resources via HTTP.
type HTTP struct {
	URL        string
	Header     http.Header
	Method     string
	HTTPClient httpClient
}

func (loader *HTTP) String() string {
	return loader.URL
}

func (loader *HTTP) Location() string {
	return loader.URL
}

func (loader *HTTP) Open() (reader io.ReadCloser, err error) {
	method := loader.Method
	if len(method) == 0 {
		method = DefaultMethod
	}

	var (
		request  *http.Request
		response *http.Response
	)

	request, err = http.NewRequest(method, loader.URL, nil)
	if err != nil {
		return
	}

	for key, values := range loader.Header {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	HTTPClient := loader.HTTPClient
	if HTTPClient == nil {
		HTTPClient = http.DefaultClient
	}

	response, err = HTTPClient.Do(request)
	defer func() {
		if err != nil && response != nil && response.Body != nil {
			response.Body.Close()
		}
	}()

	if err != nil {
		return
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		err = fmt.Errorf(
			"Unable to access [%s]: server returned %s",
			loader.URL,
			response.Status,
		)

		return
	}

	reader = response.Body
	return
}

// File is a Loader which obtains resources from the filesystem
type File struct {
	Path string
}

func (loader *File) String() string {
	return loader.Path
}

func (loader *File) Location() string {
	return loader.Path
}

func (loader *File) Open() (reader io.ReadCloser, err error) {
	reader, err = os.Open(loader.Path)
	if err != nil && reader != nil {
		reader.Close()
		reader = nil
	}

	return
}

// Data is an in-memory resource.  It is a Loader which simple reads from
// a byte slice.
type Data struct {
	Source []byte
}

func (loader *Data) String() string {
	return string(loader.Source)
}

func (loader *Data) Location() string {
	return string(loader.Source)
}

func (loader *Data) Open() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(loader.Source)), nil
}
