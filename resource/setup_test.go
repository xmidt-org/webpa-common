package resource

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	fileNameParameter = "filename"
	testFile          = "testfile.txt"
	testContents      = "here is a lovely little test file"
)

var (
	currentDirectory string
	httpServer       *httptest.Server

	testFilePath string
	testFileURL  string
)

type readCloser struct {
	reader io.Reader
	closed bool
}

func (r *readCloser) Read(buffer []byte) (int, error) {
	if r.closed {
		return 0, errors.New("ReadCloser has been closed")
	}

	return r.reader.Read(buffer)
}

func (r *readCloser) Close() error {
	if r.closed {
		return errors.New("ReadCloser already closed")
	}

	r.closed = true
	return nil
}

type testHTTPClient struct {
	transport func(*http.Request) (*http.Response, error)
}

func (c *testHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return c.transport(request)
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		var err error
		currentDirectory, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get current directory: %v\n", err)
			return 1
		}

		httpServer = httptest.NewServer(
			http.FileServer(
				http.Dir(currentDirectory),
			),
		)

		defer httpServer.Close()

		testFilePath = fmt.Sprintf("%s/%s", currentDirectory, testFile)
		testFileURL = fmt.Sprintf("%s/%s", httpServer.URL, testFile)

		fmt.Printf("Running test HTTP server at: %s\n", httpServer.URL)
		return m.Run()
	}())
}
