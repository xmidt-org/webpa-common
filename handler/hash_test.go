package handler

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/canonical"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"testing"
)

// testServiceHash just returns the same, expected value for all keys
type testServiceHash struct {
	wasCalled bool
	value     string
}

func (t *testServiceHash) Get([]byte) (string, error) {
	t.wasCalled = true
	return t.value, nil
}

// errorServiceHash always returns an error for any key
type errorServiceHash struct {
	wasCalled   bool
	errorString string
}

func (e *errorServiceHash) Get([]byte) (string, error) {
	e.wasCalled = true
	return "", errors.New(e.errorString)
}

func ExampleHash() {
	var output bytes.Buffer
	logger := &logging.LoggerWriter{&output}
	ctx := fact.SetLogger(context.Background(), logger)

	deviceId, err := canonical.ParseId("mac:111122223333")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing device id: %v\n", err)
		return
	}

	ctx = fact.SetDeviceId(ctx, deviceId)
	serviceHash := &testServiceHash{value: "http://comcast.net"}
	response, request := dummyHttpOperation()
	request.URL.Path = "/foo/bar"
	hash := Hash(serviceHash)

	hash.ServeHTTP(ctx, response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Header().Get("Location"))

	// Output:
	// 307
	// http://comcast.net/foo/bar
}

func TestHashCustomSuccess(t *testing.T) {
	const (
		service    = "http://comcast.net"
		path       = "/test/path"
		deviceName = "mac:111122223333"
	)

	assert := assert.New(t)
	deviceId, err := canonical.ParseId(deviceName)
	if !assert.Nil(err) {
		return
	}

	ctx := fact.SetDeviceId(context.Background(), deviceId)
	serviceHash := &testServiceHash{value: service}
	response, request := dummyHttpOperation()
	request.URL.Path = path
	hash := HashCustom(serviceHash, 457)
	hash.ServeHTTP(ctx, response, request)

	assert.True(serviceHash.wasCalled)
	assert.Equal(457, response.Code)
	assert.Equal(response.Header().Get("Location"), service+path)
}

func TestHashCustomNoNodes(t *testing.T) {
	const (
		path       = "/test/path"
		deviceName = "mac:111122223333"
	)

	assert := assert.New(t)
	deviceId, err := canonical.ParseId(deviceName)
	if !assert.Nil(err) {
		return
	}

	ctx := fact.SetDeviceId(context.Background(), deviceId)
	serviceHash := &errorServiceHash{errorString: "expected"}
	response, request := dummyHttpOperation()
	request.URL.Path = path
	hash := HashCustom(serviceHash, 457)
	hash.ServeHTTP(ctx, response, request)

	assert.True(serviceHash.wasCalled)
	assertJsonErrorResponse(assert, response, http.StatusServiceUnavailable, "No nodes available: "+serviceHash.errorString)
}
