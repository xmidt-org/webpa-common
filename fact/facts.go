package fact

import (
	"errors"
	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"golang.org/x/net/context"
)

const (
	loggerKey int = iota
	deviceIdKey
	conveyKey
	tokenKey
)

var (
	NoLogger   = errors.New("No Logger found in context")
	NoDeviceId = errors.New("No deviceId found in context")
	NoConvey   = errors.New("No convey payload found in context")
	NoToken    = errors.New("No secure token found in context")
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

// DeviceId retrieves the device.ID of the device from the enclosing context
func DeviceId(ctx context.Context) (device.ID, bool) {
	value, ok := ctx.Value(deviceIdKey).(device.ID)
	return value, ok
}

// MustDeviceId retrieves the device.ID from the context, panicking
// if no such device id is found.
func MustDeviceId(ctx context.Context) device.ID {
	value, ok := DeviceId(ctx)
	if !ok {
		panic(NoDeviceId)
	}

	return value
}

// SetDeviceId sets the device.ID of the device in the enclosing context
func SetDeviceId(ctx context.Context, value device.ID) context.Context {
	return context.WithValue(ctx, deviceIdKey, value)
}

// Convey retrieves the device.Convey from the enclosing context
func Convey(ctx context.Context) (device.Convey, bool) {
	value, ok := ctx.Value(conveyKey).(device.Convey)
	return value, ok
}

// MustConvey retrieves the convey payload from the context, panicking
// if no such payload is found.
func MustConvey(ctx context.Context) device.Convey {
	value, ok := Convey(ctx)
	if !ok {
		panic(NoConvey)
	}

	return value
}

// SetConvey sets the device.Convey in the enclosing context
func SetConvey(ctx context.Context, value device.Convey) context.Context {
	return context.WithValue(ctx, conveyKey, value)
}

// Token retrieves the secure.Token from the enclosing context
func Token(ctx context.Context) (*secure.Token, bool) {
	value, ok := ctx.Value(tokenKey).(*secure.Token)
	return value, ok
}

// MustToken retrieves the secure token from the context, panicking
// if no such token is found.
func MustToken(ctx context.Context) *secure.Token {
	value, ok := Token(ctx)
	if !ok {
		panic(NoToken)
	}

	return value
}

// SetToken sets the secure.Token in the enclosing context
func SetToken(ctx context.Context, value *secure.Token) context.Context {
	return context.WithValue(ctx, tokenKey, value)
}
