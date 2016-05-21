package fact

import (
	"errors"
	"github.com/Comcast/webpa-common/canonical"
	"github.com/Comcast/webpa-common/convey"
	"github.com/Comcast/webpa-common/logging"
	"golang.org/x/net/context"
)

const (
	loggerKey int = iota
	deviceIdKey
	conveyKey
)

var (
	NoLogger   = errors.New("No Logger found in context")
	NoDeviceId = errors.New("No deviceId found in context")
	NoConvey   = errors.New("No convey payload found in context")
)

// Logger retrieves the logging.Logger from the enclosing context
func Logger(ctx context.Context) (logging.Logger, bool) {
	value, ok := ctx.Value(loggerKey).(logging.Logger)
	return value, ok
}

// MustLogger retrieves the logging.Logger from the context, panicking
// if no such logger is found.
func MustLogger(ctx context.Context) logging.Logger {
	value, ok := Logger(ctx)
	if !ok {
		panic(NoLogger)
	}

	return value
}

// SetLogger sets the logging.Logger in the enclosing context
func SetLogger(ctx context.Context, value logging.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, value)
}

// DeviceId retrieves the canonical.Id of the device from the enclosing context
func DeviceId(ctx context.Context) (canonical.Id, bool) {
	value, ok := ctx.Value(deviceIdKey).(canonical.Id)
	return value, ok
}

// MustDeviceId retrieves the canonical.Id from the context, panicking
// if no such device id is found.
func MustDeviceId(ctx context.Context) canonical.Id {
	value, ok := DeviceId(ctx)
	if !ok {
		panic(NoDeviceId)
	}

	return value
}

// SetDeviceId sets the canonical.Id of the device in the enclosing context
func SetDeviceId(ctx context.Context, value canonical.Id) context.Context {
	return context.WithValue(ctx, deviceIdKey, value)
}

// Convey retrieves the convey.Payload from the enclosing context
func Convey(ctx context.Context) (convey.Payload, bool) {
	value, ok := ctx.Value(conveyKey).(convey.Payload)
	return value, ok
}

// MustConvey retrieves the convey payload from the context, panicking
// if no such payload is found.
func MustConvey(ctx context.Context) convey.Payload {
	value, ok := Convey(ctx)
	if !ok {
		panic(NoConvey)
	}

	return value
}

// SetConvey sets the convey.Payload in the enclosing context
func SetConvey(ctx context.Context, value convey.Payload) context.Context {
	return context.WithValue(ctx, conveyKey, value)
}
