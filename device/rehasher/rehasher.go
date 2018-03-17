package rehasher

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/monitor"
)

// Option is a configuration option for a rehasher
type Option func(*rehasher)

// WithLogger configures a rehasher with a logger, using the default logger if l is nil.
func WithLogger(l log.Logger) Option {
	return func(r *rehasher) {
		if l == nil {
			r.logger = logging.DefaultLogger()
		} else {
			r.logger = l
		}
	}
}

// WithAccessorFactory configures a rehasher with a specific factory for service.Accessor objects.
// If af is nil, the default accessor factory is used.
func WithAccessorFactory(af service.AccessorFactory) Option {
	return func(r *rehasher) {
		if af == nil {
			r.accessorFactory = service.DefaultAccessorFactory
		} else {
			r.accessorFactory = af
		}
	}
}

// WithIsRegistered configures a rehasher with a strategy for determining if a discovered service instance
// is registered as this process.  There is no default.  If nil, rehashing is disabled.
func WithIsRegistered(f func(string) bool) Option {
	return func(r *rehasher) {
		r.isRegistered = f
	}
}

// WithEnvironment configures a rehasher to use a service discovery environment.  If non-nil, the given environment's
// accessor factory and IsRegistered strategy are used.  If nil, rehashing is disabled and the default
// accessor factory is used.
func WithEnvironment(e service.Environment) Option {
	return func(r *rehasher) {
		if e == nil {
			r.accessorFactory = service.DefaultAccessorFactory
			r.isRegistered = nil
		} else {
			r.accessorFactory = e.AccessorFactory()
			r.isRegistered = e.IsRegistered
		}
	}
}

// New creates a monitor Listener which will rehash and disconnect devices in response to service discovery events.
// This function panics if the connector is nil.
//
// If the returned listener encounters any service discovery error, all devices are disconnected.  If an IsRegistered
// strategy is configured, typically via WithEnvironment, then devices that hash to an instance that is not this process
// will be disconnected in response to service discovery updates.
func New(c device.Connector, options ...Option) monitor.Listener {
	if c == nil {
		panic("A device Connector is required")
	}

	r := &rehasher{
		logger:          logging.DefaultLogger(),
		accessorFactory: service.DefaultAccessorFactory,
		connector:       c,
	}

	for _, o := range options {
		o(r)
	}

	return r
}

// rehasher implements monitor.Listener and (1) disconnects all devices when any service discovery error occurs,
// and (2) optionally rehashes devices in response to updated instances.
type rehasher struct {
	logger          log.Logger
	accessorFactory service.AccessorFactory
	isRegistered    func(string) bool
	connector       device.Connector
}

func (r *rehasher) MonitorEvent(e monitor.Event) {
	switch {
	case e.Err != nil:
		r.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery error", logging.ErrorKey(), e.Err)
		r.connector.DisconnectAll()

	case e.Stopped:
		r.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery monitor being stopped")
		r.connector.DisconnectAll()

	case len(e.Instances) > 0:
		r.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehashing devices", "instances", e.Instances)

		a := r.accessorFactory(e.Instances)
		disconnectCount := r.connector.DisconnectIf(func(id device.ID) bool {
			instance, err := a.Get(id.Bytes())
			if err != nil {
				r.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting device: error during rehash", logging.ErrorKey(), err, "id", id)
				return true
			}

			if !r.isRegistered(instance) {
				r.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting device: rehashed to another instance", "instance", instance, "id", id)
				return true
			}

			return true
		})

		r.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash complete", "disconnectCount", disconnectCount)

	default:
		r.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery updated with no instances")
		r.connector.DisconnectAll()
	}
}
