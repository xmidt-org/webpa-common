package device

import (
	"github.com/Comcast/webpa-common/logging"
	"time"
)

const (
	DefaultDeviceNameHeader = "X-Webpa-Device-Name"
	DefaultConveyHeader     = "X-Webpa-Convey"

	DefaultHandshakeTimeout time.Duration = 10 * time.Second
	DefaultIdlePeriod       time.Duration = 135 * time.Second
	DefaultWriteTimeout     time.Duration = 60 * time.Second
	DefaultPingPeriod       time.Duration = 45 * time.Second

	DefaultInitialCapacity        = 100000
	DefaultReadBufferSize         = 4096
	DefaultWriteBufferSize        = 4096
	DefaultDeviceMessageQueueSize = 100
)

// Options represent the available configuration options for components
// within this package
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

	// InitialCapacity is used as the starting capacity of the internal map of
	// registered devices.  If not supplied, DefaultInitialCapacity is used.
	InitialCapacity int

	// ReadBufferSize is the optional size of websocket read buffers.  If not supplied,
	// the internal gorilla default is used.
	ReadBufferSize int

	// WriteBufferSize is the optional size of websocket write buffers.  If not supplied,
	// the internal gorilla default is used.
	WriteBufferSize int

	// Subprotocols is the optional slice of websocket subprotocols to use.
	Subprotocols []string

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

	// MessageReceivedListener is the notification sink for device messages
	MessageReceivedListener MessageReceivedListener

	// ConnectListener receives notifications for device connections
	ConnectListener ConnectListener

	// DisconnectListener receives notifications when devices disconnect for any reason
	DisconnectListener DisconnectListener

	// PongListener is the notification sink for pongs
	PongListener PongListener

	// KeyFunc is the factory function for Keys, used when devices connect.
	// If this value is nil, then UUIDKeyFunc is used along with crypto/rand's Reader.
	KeyFunc KeyFunc

	// Logger is the output sink for log messages.  If not supplied, log output
	// is sent to logging.DefaultLogger().
	Logger logging.Logger
}

func (o *Options) deviceNameHeader() string {
	if o != nil && len(o.DeviceNameHeader) > 0 {
		return o.DeviceNameHeader
	}

	return DefaultDeviceNameHeader
}

func (o *Options) conveyHeader() string {
	if o != nil && len(o.ConveyHeader) > 0 {
		return o.ConveyHeader
	}

	return DefaultConveyHeader
}

func (o *Options) deviceMessageQueueSize() int {
	if o != nil && o.DeviceMessageQueueSize > 0 {
		return o.DeviceMessageQueueSize
	}

	return DefaultDeviceMessageQueueSize
}

func (o *Options) handshakeTimeout() time.Duration {
	if o != nil && o.HandshakeTimeout > 0 {
		return o.HandshakeTimeout
	}

	return DefaultHandshakeTimeout
}

func (o *Options) initialCapacity() int {
	if o != nil && o.InitialCapacity > 0 {
		return o.InitialCapacity
	}

	return DefaultInitialCapacity
}

func (o *Options) idlePeriod() time.Duration {
	if o != nil && o.IdlePeriod > 0 {
		return o.IdlePeriod
	}

	return DefaultIdlePeriod
}

func (o *Options) pingPeriod() time.Duration {
	if o != nil && o.PingPeriod > 0 {
		return o.PingPeriod
	}

	return DefaultPingPeriod
}

func (o *Options) writeTimeout() time.Duration {
	if o != nil && o.WriteTimeout > 0 {
		return o.WriteTimeout
	}

	return DefaultWriteTimeout
}

func (o *Options) readBufferSize() int {
	if o != nil && o.ReadBufferSize > 0 {
		return o.ReadBufferSize
	}

	return DefaultReadBufferSize
}

func (o *Options) writeBufferSize() int {
	if o != nil && o.WriteBufferSize > 0 {
		return o.WriteBufferSize
	}

	return DefaultWriteBufferSize
}

func (o *Options) subprotocols() (subprotocols []string) {
	if o != nil && len(o.Subprotocols) > 0 {
		subprotocols = make([]string, len(o.Subprotocols))
		copy(subprotocols, o.Subprotocols)
	}

	return
}

func (o *Options) keyFunc() KeyFunc {
	if o != nil && o.KeyFunc != nil {
		return o.KeyFunc
	}

	return UUIDKeyFunc(nil, nil)
}

func (o *Options) logger() logging.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}

func (o *Options) messageReceivedListener() MessageReceivedListener {
	if o != nil && o.MessageReceivedListener != nil {
		return o.MessageReceivedListener
	}

	return defaultMessageReceivedListener
}

func (o *Options) connectListener() ConnectListener {
	if o != nil && o.ConnectListener != nil {
		return o.ConnectListener
	}

	return defaultConnectListener
}

func (o *Options) disconnectListener() DisconnectListener {
	if o != nil && o.DisconnectListener != nil {
		return o.DisconnectListener
	}

	return defaultDisconnectListener
}

func (o *Options) pongListener() PongListener {
	if o != nil && o.PongListener != nil {
		return o.PongListener
	}

	return defaultPongListener
}
