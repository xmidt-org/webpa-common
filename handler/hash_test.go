package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
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

	deviceID, err := device.ParseID("mac:111122223333")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing device id: %v\n", err)
		return
	}

	ctx = fact.SetDeviceId(ctx, deviceID)
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
	deviceID, err := device.ParseID(deviceName)
	if !assert.Nil(err) {
		return
	}

	ctx := fact.SetDeviceId(context.Background(), deviceID)
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
	deviceID, err := device.ParseID(deviceName)
	if !assert.Nil(err) {
		return
	}

	ctx := fact.SetDeviceId(context.Background(), deviceID)
	serviceHash := &errorServiceHash{errorString: "expected"}
	response, request := dummyHttpOperation()
	request.URL.Path = path
	hash := HashCustom(serviceHash, 457)
	hash.ServeHTTP(ctx, response, request)

	assert.True(serviceHash.wasCalled)
	assertJsonErrorResponse(assert, response, http.StatusServiceUnavailable, "No nodes available: "+serviceHash.errorString)
}
