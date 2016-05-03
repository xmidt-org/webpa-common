// Package context provides the standard contextual request data for all WebPA servers
package context

import (
	"fmt"
	"github.com/Comcast/webpa-common/canonical"
	"net/http"
)

// Context is the core type of this package.
type Context interface {
	// Logger returns the contextual Logger.  It is never nil.
	Logger() Logger

	// DeviceId returns the canonical device id associated with the request.
	DeviceId() canonical.Id
}

// defaultContext is the default implementation of Context
type defaultContext struct {
	logger   Logger
	deviceId canonical.Id
}

func (c *defaultContext) Logger() Logger {
	return c.logger
}

func (c *defaultContext) DeviceId() canonical.Id {
	return c.deviceId
}

// NewContext creates a new Context object from an HTTP request
func NewContext(logger Logger, request *http.Request) (Context, error) {
	deviceName := request.Header.Get(DeviceNameHeader)
	if len(deviceName) == 0 {
		return nil, missingDeviceNameError
	}

	deviceId, err := canonical.ParseId(deviceName)
	if err != nil {
		return nil, NewHttpError(
			http.StatusBadRequest,
			fmt.Sprintf(InvalidDeviceNameHeaderPattern, deviceName),
		)
	}

	return &defaultContext{
		logger:   logger,
		deviceId: deviceId,
	}, nil
}
