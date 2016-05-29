package key

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

const (
	publicKeyFile  string = "testkey.pub"
	privateKeyFile string = "testkey"
)

var (
	httpServer *httptest.Server

	publicKeyURL  *url.URL
	publicKeyPath string

	privateKeyURL  *url.URL
	privateKeyPath string
)

func TestMain(m *testing.M) {
	testDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, "Unable to obtain current working directory: %v\n", err)
		os.Exit(1)
	}

	httpServer = httptest.NewServer(http.FileServer(http.Dir(testDirectory)))

	fmt.Printf("started test server at %s\n", httpServer.URL)
	publicKeyURL, err = url.Parse(httpServer.URL + "/" + publicKeyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse public key URL: %v\n")
		os.Exit(1)
	}

	publicKeyPath = filepath.Join(testDirectory, publicKeyFile)
	fmt.Printf("test public key URL: %s\n", publicKeyURL)
	fmt.Printf("test public key path: %s\n", publicKeyPath)

	privateKeyURL, err = url.Parse(httpServer.URL + "/" + privateKeyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse private key URL: %v\n")
		os.Exit(1)
	}

	privateKeyPath = filepath.Join(testDirectory, privateKeyFile)
	fmt.Printf("test private key URL: %s\n", privateKeyURL)
	fmt.Printf("test private key path: %s\n", privateKeyPath)

	defer httpServer.Close()
	os.Exit(m.Run())
}
