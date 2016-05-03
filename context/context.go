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

	// Get returns the value of a specific attribute.
	Get(attributeName string) interface{}

	// GetOk is like Get, but it returns a second parameter indicating whether
	// the attribute exists.
	GetOk(attributeName string) (interface{}, bool)
}

// defaultContext is the default implementation of Context
type defaultContext struct {
	logger     Logger
	deviceId   canonical.Id
	attributes map[string]interface{}
}

func (c *defaultContext) Logger() Logger {
	return c.logger
}

func (c *defaultContext) DeviceId() canonical.Id {
	return c.deviceId
}

func (c *defaultContext) Get(attributeName string) interface{} {
	return c.attributes[attributeName]
}

func (c *defaultContext) Get(attributeName string) (interface{}, bool) {
	value, ok := c.attributes[attributeName]
	return value, ok
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
		logger:     logger,
		deviceId:   deviceId,
		attributes: make(map[string]interface{}, 10),
	}, nil
}
