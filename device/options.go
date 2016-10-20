package device

import (
	"github.com/Comcast/webpa-common/logging"
	"time"
)

const (
	DefaultInitialRegistrySize    = 10000
	DefaultDeviceMessageQueueSize = 100

	DefaultPingPeriod   time.Duration = 45 * time.Second
	DefaultIdlePeriod   time.Duration = 135 * time.Second
	DefaultWriteTimeout time.Duration = 60 * time.Second
)

var (
	// defaultOptions is the internal Options instance used for default values.
	// Useful when a nil Options is passed to something.
	defaultOptions = Options{}
)

// Options represent the available configuration options for device Managers
type Options struct {
	// DeviceNameHeader is the name of the HTTP request header which contains the
	// device name.  If not specified, DefaultDeviceNameHeader is used.
	DeviceNameHeader string

	// ConveyHeader is the name of the HTTP request header which contains the
	// base64-encoded JSON payload to forward with each outbound device request.
	// If not specified, DefaultConveyHeader is used.
	ConveyHeader string

	// HandshakeTimeout is the optional websocket handshake timeout.  If not supplied,
	// the internal gorilla default is used.
	HandshakeTimeout time.Duration

	// ReadBufferSize is the optional size of websocket read buffers.  If not supplied,
	// the internal gorilla default is used.
	ReadBufferSize int

	// WriteBufferSize is the optional size of websocket write buffers.  If not supplied,
	// the internal gorilla default is used.
	WriteBufferSize int

	// Subprotocols is the optional slice of websocket subprotocols to use.
	Subprotocols []string

	// InitialRegistrySize is the initial capacity of the internal map of devices.
	// If not supplied, DefaultInitialRegistrySize is used.
	InitialRegistrySize int

	// DeviceMessageQueueSize is the capacity of the channel which stores messages waiting
	// to be transmitted to a device.  If not supplied, DefaultDeviceMessageQueueSize is used.
	DeviceMessageQueueSize int

	// PingPeriod is the time between pings sent to each device
	PingPeriod time.Duration

	// IdlePeriod is the length of time a device connection is allowed to be idle,
	// with no traffic coming from the device.  If not supplied, DefaultIdlePeriod is used.
	IdlePeriod time.Duration

	// WriteTimeout is the write timeout for each device's websocket.  If not supplied,
	// DefaultWriteTimeout is used.
	WriteTimeout time.Duration

	// Listeners is the aggregate set of listeners which receive events from
	// the constructed Manager.
	Listeners Listeners

	// Logger is the output sink for log messages.  If not supplied, log output
	// is sent to logging.DefaultLogger().
	Logger logging.Logger
}

func (o *Options) initialRegistrySize() int {
	if o.InitialRegistrySize > 0 {
		return o.InitialRegistrySize
	}

	return DefaultInitialRegistrySize
}

func (o *Options) pingPeriod() time.Duration {
	if o.PingPeriod > 0 {
		return o.PingPeriod
	}

	return DefaultPingPeriod
}

func (o *Options) subprotocols() []string {
	if len(o.Subprotocols) == 0 {
		return nil
	}

	subprotocols := make([]string, len(o.Subprotocols))
	copy(subprotocols, o.Subprotocols)
	return subprotocols
}

func (o *Options) deviceMessageQueueSize() int {
	if o.DeviceMessageQueueSize > 0 {
		return o.DeviceMessageQueueSize
	}

	return DefaultDeviceMessageQueueSize
}

func (o *Options) logger() logging.Logger {
	if o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}
