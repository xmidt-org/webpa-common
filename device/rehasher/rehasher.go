package rehasher

import (
	"time"

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
// is registered as this process.  There is no default.
func WithIsRegistered(f func(string) bool) Option {
	return func(r *rehasher) {
		r.isRegistered = f
	}
}

// WithEnvironment configures a rehasher to use a service discovery environment.
func WithEnvironment(e service.Environment) Option {
	return func(r *rehasher) {
		r.accessorFactory = e.AccessorFactory()
		r.isRegistered = e.IsRegistered
	}
}

// New creates a monitor Listener which will rehash and disconnect devices in response to service discovery events.
// This function panics if the connector is nil or if no IsRegistered strategy is configured.
//
// If the returned listener encounters any service discovery error, all devices are disconnected.  Otherwise,
// the IsRegistered strategy is used to determine which devices should still be connected to the Connector.  Devices
// that hash to instances not registered in this environment are disconnected.
func New(connector device.Connector, options ...Option) monitor.Listener {
	if connector == nil {
		panic("A device Connector is required")
	}

	r := &rehasher{
		logger:          logging.DefaultLogger(),
		accessorFactory: service.DefaultAccessorFactory,
		connector:       connector,
		now:             time.Now,
	}

	for _, o := range options {
		o(r)
	}

	if r.isRegistered == nil {
		panic("No IsRegistered strategy configured.  Use WithIsRegistered or WithEnvironment.")
	}

	return r
}

// rehasher implements monitor.Listener and (1) disconnects all devices when any service discovery error occurs,
// and (2) rehashes devices in response to updated instances.
type rehasher struct {
	logger          log.Logger
	accessorFactory service.AccessorFactory
	isRegistered    func(string) bool
	connector       device.Connector
	now             func() time.Time
}

func (r *rehasher) rehash(logger log.Logger, accessor service.Accessor) {
	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash starting")

	var (
		start = r.now()

		disconnectCount = r.connector.DisconnectIf(func(candidate device.ID) bool {
			instance, err := accessor.Get(candidate.Bytes())
			switch {
			case err != nil:
				logger.Log(level.Key(), level.ErrorValue(),
					logging.MessageKey(), "disconnecting device: error during rehash",
					logging.ErrorKey(), err,
					"id", candidate,
				)

				return true

			case !r.isRegistered(instance):
				logger.Log(level.Key(), level.InfoValue(),
					logging.MessageKey(), "disconnecting device: rehashed to another instance",
					"instance", instance,
					"id", candidate,
				)

				return true

			default:
				logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "device hashed to this instance", "id", candidate)
				return false
			}
		})

		elapsed = r.now().Sub(start)
	)

	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash complete", "disconnectCount", disconnectCount, "elapsed", elapsed)
}

func (r *rehasher) MonitorEvent(e monitor.Event) {
	logger := logging.Enrich(
		log.With(
			r.logger,
			monitor.EventCountKey(), e.EventCount,
		),
		e.Instancer,
	)

	switch {
	case e.Err != nil:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery error", logging.ErrorKey(), e.Err)
		r.connector.DisconnectAll()

	case e.Stopped:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery monitor being stopped")
		r.connector.DisconnectAll()

	case e.EventCount == 1:
		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "ignoring initial instances")

	case len(e.Instances) > 0:
		r.rehash(logger, r.accessorFactory(e.Instances))

	default:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery updated with no instances")
		r.connector.DisconnectAll()
	}
}
