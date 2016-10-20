package device

import (
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
