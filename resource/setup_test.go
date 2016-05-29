package resource

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	testFile     string = "testfile.txt"
	testContents string = "here is a lovely little test file"
)

var (
	currentDirectory string
	httpServer       *httptest.Server
)

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

		fmt.Printf("Running test server at: %s\n", httpServer.URL)
		return m.Run()
	}())
}
