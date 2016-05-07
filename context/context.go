package context

import (
	"encoding/base64"
	"fmt"
	"github.com/Comcast/webpa-common/canonical"
	"github.com/Comcast/webpa-common/convey"
	"net/http"
)

// ConveyPayload is a map type which stores the optional, deserialized convey value
type ConveyPayload map[string]interface{}

// Context is the core type of this package.
type Context interface {
	// Logger returns the contextual Logger.  It is never nil.
	Logger() Logger

	// DeviceId returns the canonical device id associated with the request.
	DeviceId() canonical.Id

	// ConveyPayload returns the optional payload of a convey header
	ConveyPayload() convey.Payload
}

// defaultContext is the default implementation of Context
type defaultContext struct {
	logger        Logger
	deviceId      canonical.Id
	conveyPayload convey.Payload
}

func (c *defaultContext) Logger() Logger {
	return c.logger
}

func (c *defaultContext) DeviceId() canonical.Id {
	return c.deviceId
}

func (c *defaultContext) ConveyPayload() convey.Payload {
	return c.conveyPayload
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

	var conveyPayload convey.Payload
	rawConveyValue := request.Header.Get(ConveyHeader)
	if len(rawConveyValue) > 0 {
		conveyPayload, err = convey.ParsePayload(base64.StdEncoding, rawConveyValue)
		if err != nil {
			return nil, NewHttpError(
				http.StatusBadRequest,
				fmt.Sprintf(InvalidConveyPattern, rawConveyValue),
			)
		}
	}

	return &defaultContext{
		logger:        logger,
		deviceId:      deviceId,
		conveyPayload: conveyPayload,
	}, nil
}
