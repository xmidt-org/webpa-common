package key

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type keyInfo struct {
	fileName string
	public   bool
	filePath string
	fileURL  *url.URL
	key      interface{}
}

var (
	httpServer *httptest.Server
	publicKey  = keyInfo{fileName: "testkey.pub", public: true}.initialize()
	privateKey = keyInfo{fileName: "testkey", public: false}.initialize()
)

func (keyInfo keyInfo) initialize() *keyInfo {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "%v\n", r)
			os.Exit(1)
		}
	}()

	if len(keyInfo.fileName) == 0 {
		panic("Invalid test configuration")
	}

	filePath, err := filepath.Abs(keyInfo.fileName)
	if err != nil {
		panic(fmt.Sprintf("Failed to determine absolute path to %s: %v", keyInfo.fileName, err))
	}

	keyInfo.filePath = filepath.ToSlash(filePath)

	var rawFileURL string
	if filePath[0] != '/' {
		// you know, for Windows ....
		rawFileURL = "file:///" + keyInfo.filePath
	} else {
		rawFileURL = "file://" + keyInfo.filePath
	}

	keyInfo.fileURL, err = url.Parse(rawFileURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse URL %s: %v", rawFileURL, err))
	}

	rawData, err := ioutil.ReadFile(keyInfo.filePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read key file %s: %v", keyInfo.filePath, err))
	}

	block, _ := pem.Decode(rawData)
	if block == nil {
		panic(fmt.Sprintf("Failed to decode PEM block in key file %s: %v", keyInfo.filePath, err))
	}

	if keyInfo.public {
		keyInfo.key, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			panic(fmt.Sprintf("Failed to decode public key file %s: %v", keyInfo.filePath, err))
		}
	} else {
		keyInfo.key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			panic(fmt.Sprintf("Failed to decode private key file %s: %v", keyInfo.fileName, err))
		}
	}

	return &keyInfo
}

// httpURL returns the URL to the key in the test HTTP server
func (keyInfo *keyInfo) httpURL() *url.URL {
	url, err := url.Parse(httpServer.URL + "/" + keyInfo.fileName)
	if err != nil {
		panic(err)
	}

	return url
}

// loaderEqual performs a recursive comparison of two Loader objects
// cache time comparison assumes that the expected loader was created before
// the actual loader
func loaderEqual(actual, expected Loader) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()

	switch actual := actual.(type) {
	case *urlLoader:
		expected := expected.(*urlLoader)
		if actual.name != expected.name {
			err = errors.New(fmt.Sprintf("Expected urlLoader.name %s but got %s", expected.name, actual.name))
		} else if actual.purpose != expected.purpose {
			err = errors.New(fmt.Sprintf("Expected urlLoader.purpose %d but got %d", expected.purpose, actual.purpose))
		} else if actual.url != expected.url {
			err = errors.New(fmt.Sprintf("Expected urlLoader.url %s but got %s", expected.url, actual.url))
		}

	case *oneTimeLoader:
		expected := expected.(*oneTimeLoader)
		err = loaderEqual(actual.delegate, expected.delegate)
		if err == nil && !reflect.DeepEqual(actual.key, expected.key) {
			err = errors.New(fmt.Sprintf("Expected one-time loader key %s but got %s", expected.key, actual.key))
		}

	case *cacheLoader:
		expected := expected.(*cacheLoader)
		err = loaderEqual(actual.delegate, expected.delegate)
		if err == nil {
			if !reflect.DeepEqual(actual.cachedKey, expected.cachedKey) {
				err = errors.New(fmt.Sprintf("Expected cached loader key %s but got %s", expected.cachedKey, actual.cachedKey))
			} else if actual.cacheExpiry.Before(expected.cacheExpiry) {
				err = errors.New(fmt.Sprintf("Expected minimum cache expiry %d but got %d", expected.cacheExpiry, actual.cacheExpiry))
			}
		}

	default:
		err = errors.New(fmt.Sprintf("Unrecognized actual type %s", reflect.TypeOf(actual)))
	}

	return
}

func resolverEqual(actual, expected Resolver) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
	}()

	switch actual := actual.(type) {
	case *mapResolver:
		expected := expected.(*mapResolver)
		_ = expected

	default:
		err = errors.New(fmt.Sprintf("Unrecognized actual type %s", reflect.TypeOf(actual)))
	}
	return
}

func TestMain(m *testing.M) {
	testDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, "Unable to obtain current working directory: %v\n", err)
		os.Exit(1)
	}

	httpServer = httptest.NewServer(http.FileServer(http.Dir(testDirectory)))
	defer httpServer.Close()

	os.Exit(m.Run())
}
