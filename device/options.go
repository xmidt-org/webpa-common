package device

import (
	"math"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/gorilla/websocket"
)

const (
	// DeviceNameHeader is the name of the HTTP header which contains the device service name.
	// This header is primarily required at connect time to identify the device.
	DeviceNameHeader = "X-Webpa-Device-Name"

	// ConveyHeader is the name of the optional HTTP header which contains the encoded convey JSON.
	ConveyHeader = "X-Webpa-Convey"

	DefaultIdlePeriod     time.Duration = 135 * time.Second
	DefaultRequestTimeout time.Duration = 30 * time.Second
	DefaultWriteTimeout   time.Duration = 60 * time.Second
	DefaultPingPeriod     time.Duration = 45 * time.Second
	DefaultAuthDelay      time.Duration = 1 * time.Second

	DefaultDecoderPoolSize        = 1000
	DefaultEncoderPoolSize        = 1000
	DefaultInitialCapacity        = 1000
	DefaultReadBufferSize         = 0
	DefaultWriteBufferSize        = 0
	DefaultDeviceMessageQueueSize = 100
)

// Options represent the available configuration options for components
// within this package
type Options struct {
	// Upgrader is the gorilla websocket.Upgrader injected into these options.
	Upgrader websocket.Upgrader

	// InitialCapacity is used as the starting capacity of the internal map of
	// registered devices.  If not supplied, DefaultInitialCapacity is used.
	InitialCapacity uint32

	// MaxDevices is the maximum number of devices allowed to connect to any one Manager.
	// If unset (i.e. zero), math.MaxUint32 is used as the maximum.
	MaxDevices uint32

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

	// Logger is the output sink for log messages.  If not supplied, log output
	// is sent to a NOP logger.
	Logger log.Logger

	// MetricsProvider is the go-kit factory for metrics
	MetricsProvider provider.Provider

	// Now is the closure used to determine the current time.  If not set, time.Now is used.
	Now func() time.Time
}

func (o *Options) upgrader() *websocket.Upgrader {
	upgrader := new(websocket.Upgrader)
	if o != nil {
		*upgrader = o.Upgrader
	}

	return upgrader
}

func (o *Options) deviceMessageQueueSize() int {
	if o != nil && o.DeviceMessageQueueSize > 0 {
		return o.DeviceMessageQueueSize
	}

	return DefaultDeviceMessageQueueSize
}

func (o *Options) initialCapacity() uint32 {
	if o != nil && o.InitialCapacity > 0 {
		return o.InitialCapacity
	}

	return DefaultInitialCapacity
}

func (o *Options) maxDevices() uint32 {
	if o != nil && o.MaxDevices > 0 {
		return o.MaxDevices
	}

	return math.MaxUint32
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

func (o *Options) logger() log.Logger {
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

func (o *Options) metricsProvider() provider.Provider {
	if o != nil && o.MetricsProvider != nil {
		return o.MetricsProvider
	}

	return provider.NewDiscardProvider()
}

func (o *Options) now() func() time.Time {
	if o != nil && o.Now != nil {
		return o.Now
	}

	return time.Now
}
