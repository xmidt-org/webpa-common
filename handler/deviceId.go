package handler

import (
	"fmt"
	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/fact"
	"golang.org/x/net/context"
	"net/http"
)

// DeviceId parses out the DeviceNameHeader from the request
func DeviceId() ChainHandler {
	return DeviceIdCustom(DeviceNameHeader)
}

// DeviceIdCustom allows the header used for the device name to be customized
func DeviceIdCustom(deviceNameHeader string) ChainHandler {
	return ChainHandlerFunc(func(ctx context.Context, response http.ResponseWriter, request *http.Request, next ContextHandler) {
		deviceName := request.Header.Get(deviceNameHeader)
		if len(deviceName) == 0 {
			WriteJsonError(response, http.StatusBadRequest, MissingDeviceNameHeaderMessage)
			return
		}

		deviceId, err := canonical.ParseId(deviceName)
		if err != nil {
			WriteJsonError(
				response,
				http.StatusBadRequest,
				fmt.Sprintf(InvalidDeviceNameHeaderPattern, deviceName, err),
			)

			return
		}

		next.ServeHTTP(
			fact.SetDeviceId(ctx, deviceId),
			response,
			request,
		)
	})
}
