package device

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// NewConnectHandler produces an http.Handler that allows devices to connect
// to a specific Manager.
func NewConnectHandler(manager Manager, logger logging.Logger) http.Handler {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		device, err := manager.Connect(response, request)
		if err != nil {
			logger.Error("Failed to connect device: %s", err)
		} else {
			logger.Debug("Connected device: %s", device.ID())
		}
	})
}

// NewDeviceListHandler returns an http.Handler that renders a JSON listing
// of the devices within a manager.
func NewDeviceListHandler(manager Manager, timeLayout string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		devices := make(map[ID][]map[string]interface{})
		manager.VisitAll(func(device Interface) {
			entry := map[string]interface{}{
				"connectedAt": device.ConnectedAt().Format(timeLayout),
			}

			convey := device.Convey()
			if convey != nil && len(convey.decoded) > 0 {
				entry["convey"] = convey.decoded
			}

			key := device.ID()
			devices[key] = append(devices[key], entry)
		})

		data, err := json.Marshal(devices)
		if err != nil {
			httperror.Write(response, err)
			return
		}

		response.Header().Set("Content-Type", "application/json")
		response.Write(data)
	})
}
