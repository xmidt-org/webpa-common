package device

import (
	"time"

	"github.com/Comcast/webpa-common/logging"
)

const (
	// DeviceNameHeader is the name of the HTTP header which contains the device service name.
	// This header is primarily required at connect time to identify the device.
	DeviceNameHeader = "X-Webpa-Device-Name"

	// ConveyHeader is the name of the optional HTTP header which contains the encoded convey JSON.
	ConveyHeader = "X-Webpa-Convey"

	DefaultHandshakeTimeout time.Duration = 10 * time.Second
	DefaultIdlePeriod       time.Duration = 135 * time.Second
	DefaultRequestTimeout   time.Duration = 30 * time.Second
	DefaultWriteTimeout     time.Duration = 60 * time.Second
	DefaultPingPeriod       time.Duration = 45 * time.Second
	DefaultAuthDelay        time.Duration = 1 * time.Second

	DefaultDecoderPoolSize        = 1000
	DefaultEncoderPoolSize        = 1000
	DefaultInitialCapacity        = 1000
	DefaultReadBufferSize         = 4096
	DefaultWriteBufferSize        = 4096
	DefaultDeviceMessageQueueSize = 100
)

// Options represent the available configuration options for components
// within this package
type Options struct {
	// HandshakeTimeout is the optional websocket handshake timeout.  If not supplied,
	// the internal gorilla default is used.
	HandshakeTimeout time.Duration

	// DecoderPoolSize is the size of the pool of wrp.Decoder objects used internally
	// to decode messages from external sources, such as HTTP requests
	DecoderPoolSize int

	// EncoderPoolSize is the size of the pool of wrp.Encoder objects used internally
	// to encode messages that have no encoded byte representation.
	EncoderPoolSize int

	// InitialCapacity is used as the starting capacity of the internal map of
	// registered devices.  If not supplied, DefaultInitialCapacity is used.
	InitialCapacity uint32

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

	// AuthDelay is the time to wait before sending the authorization message
	AuthDelay time.Duration

	// IdlePeriod is the length of time a device connection is allowed to be idle,
	// with no traffic coming from the device.  If not supplied, DefaultIdlePeriod is used.
	IdlePeriod time.Duration

	// RequestTimeout is the timeout for all inbound HTTP requests
	RequestTimeout time.Duration

	// WriteTimeout is the write timeout for each device's websocket.  If not supplied,
	// DefaultWriteTimeout is used.
	WriteTimeout time.Duration

	// Listeners contains the event sinks for managers created using these options
	Listeners []Listener

	// KeyFunc is the factory function for Keys, used when devices connect.
	// If this value is nil, then UUIDKeyFunc is used along with crypto/rand's Reader.
	KeyFunc KeyFunc

	// Logger is the output sink for log messages.  If not supplied, log output
	// is sent to logging.DefaultLogger().
	Logger logging.Logger
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

func (o *Options) decoderPoolSize() int {
	if o != nil && o.DecoderPoolSize > 0 {
		return o.DecoderPoolSize
	}

	return DefaultDecoderPoolSize
}

func (o *Options) encoderPoolSize() int {
	if o != nil && o.EncoderPoolSize > 0 {
		return o.EncoderPoolSize
	}

	return DefaultEncoderPoolSize
}

func (o *Options) initialCapacity() uint32 {
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

func (o *Options) authDelay() time.Duration {
	if o != nil && o.AuthDelay > 0 {
		return o.AuthDelay
	}

	return DefaultAuthDelay
}

func (o *Options) requestTimeout() time.Duration {
	if o != nil && o.RequestTimeout > 0 {
		return o.RequestTimeout
	}

	return DefaultRequestTimeout
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

func (o *Options) listeners() []Listener {
	if o != nil {
		return o.Listeners
	}

	return nil
}
