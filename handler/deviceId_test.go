package handler

import (
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"testing"
)

func ExampleDeviceId() {
	logger := &logging.LoggerWriter{os.Stdout}
	ctx := fact.SetLogger(context.Background(), logger)
	contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		deviceId := fact.MustDeviceId(ctx)
		fact.MustLogger(ctx).Info("%s", deviceId)
	})

	response, request := dummyHttpOperation()
	request.Header.Add(DeviceNameHeader, "mac:111122223333")
	DeviceId().ServeHTTP(ctx, response, request, contextHandler)

	// Output: [INFO]  mac:111122223333
}

func TestDeviceId(t *testing.T) {
	assert := assert.New(t)
	const expectedDeviceId string = "mac:111122223333"

	for _, headerName := range []string{DeviceNameHeader, "X-Some-Other-Header"} {
		contextHandlerCalled := false
		contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
			contextHandlerCalled = true
			deviceId := fact.MustDeviceId(ctx)
			assert.Equal(expectedDeviceId, string(deviceId.Bytes()))
		})

		response, request := dummyHttpOperation()
		request.Header.Add(headerName, expectedDeviceId)
		DeviceIdCustom(headerName).ServeHTTP(context.Background(), response, request, contextHandler)
		assert.Equal(200, response.Code)
		assert.True(contextHandlerCalled)
	}
}
