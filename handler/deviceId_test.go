package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/fact"
	"github.com/Comcast/webpa-common/logging"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
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

	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create request: %v\n", err)
		return
	}

	request.Header.Add(DeviceNameHeader, "mac:111122223333")
	DeviceId().ServeHTTP(ctx, response, request, contextHandler)

	// Output: [INFO]  mac:111122223333
}

func TestDeviceId(t *testing.T) {
	const expectedDeviceId string = "mac:111122223333"

	for _, headerName := range []string{DeviceNameHeader, "X-Some-Other-Header"} {
		contextHandlerCalled := false
		contextHandler := ContextHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
			contextHandlerCalled = true
			deviceId := fact.MustDeviceId(ctx)
			if expectedDeviceId != string(deviceId.Bytes()) {
				t.Errorf("Expected device id %s, but got %s", expectedDeviceId, deviceId.Bytes())
			}
		})

		response := httptest.NewRecorder()
		request, err := http.NewRequest("GET", "", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create request: %v\n", err)
			return
		}

		request.Header.Add(headerName, expectedDeviceId)
		DeviceIdCustom(headerName).ServeHTTP(context.Background(), response, request, contextHandler)

		if response.Code != 200 {
			t.Errorf("Invalid response code %d", response.Code)
		}

		if !contextHandlerCalled {
			t.Fatal("Context handler was not called")
		}
	}
}
