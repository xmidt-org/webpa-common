package rehasher

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"

	"github.com/xmidt-org/webpa-common/device"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/service/monitor"
)

const (
	RehashError         = "rehash-error"
	RehashOtherInstance = "rehash-other-instance"

	ServiceDiscoveryError       = "service-discovery-error"
	ServiceDiscoveryStopped     = "service-discovery-stopped"
	ServiceDiscoveryNoInstances = "service-discovery-no-instances"
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

// WithMetricsProvider configures a metrics subsystem the resulting rehasher will use to track things.
// A nil provider passed to this option means to discard all metrics.
func WithMetricsProvider(p provider.Provider) Option {
	return func(r *rehasher) {
		if p == nil {
			p = provider.NewDiscardProvider()
		}

		r.keep = p.NewGauge(RehashKeepDevice)
		r.disconnect = p.NewGauge(RehashDisconnectDevice)
		r.disconnectAllCounter = p.NewCounter(RehashDisconnectAllCounter)
		r.timestamp = p.NewGauge(RehashTimestamp)
		r.duration = p.NewGauge(RehashDurationMilliseconds)
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

	var (
		defaultProvider = provider.NewDiscardProvider()

		r = &rehasher{
			logger:          logging.DefaultLogger(),
			accessorFactory: service.DefaultAccessorFactory,
			connector:       connector,
			now:             time.Now,

			keep:                 defaultProvider.NewGauge(RehashKeepDevice),
			disconnect:           defaultProvider.NewGauge(RehashDisconnectDevice),
			disconnectAllCounter: defaultProvider.NewCounter(RehashDisconnectAllCounter),
			timestamp:            defaultProvider.NewGauge(RehashTimestamp),
			duration:             defaultProvider.NewGauge(RehashDurationMilliseconds),
		}
	)

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

	keep                 metrics.Gauge
	disconnect           metrics.Gauge
	disconnectAllCounter metrics.Counter
	timestamp            metrics.Gauge
	duration             metrics.Gauge
}

func (r *rehasher) rehash(svc string, logger log.Logger, accessor service.Accessor) {
	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash starting")

	start := r.now()
	r.timestamp.With(service.ServiceLabel, svc).Set(float64(start.UTC().Unix()))

	var (
		keepCount = 0

		disconnectCount = r.connector.DisconnectIf(func(candidate device.ID) (device.CloseReason, bool) {
			instance, err := accessor.Get(candidate.Bytes())
			switch {
			case err != nil:
				logger.Log(level.Key(), level.ErrorValue(),
					logging.MessageKey(), "disconnecting device: error during rehash",
					logging.ErrorKey(), err,
					"id", candidate,
				)

				return device.CloseReason{Err: err, Text: RehashError}, true

			case !r.isRegistered(instance):
				logger.Log(level.Key(), level.InfoValue(),
					logging.MessageKey(), "disconnecting device: rehashed to another instance",
					"instance", instance,
					"id", candidate,
				)

				return device.CloseReason{Text: RehashOtherInstance}, true

			default:
				logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "device hashed to this instance", "id", candidate)
				keepCount++
				return device.CloseReason{}, false
			}
		})

		duration = r.now().Sub(start)
	)

	r.keep.With(service.ServiceLabel, svc).Set(float64(keepCount))
	r.disconnect.With(service.ServiceLabel, svc).Set(float64(disconnectCount))
	r.duration.With(service.ServiceLabel, svc).Set(float64(duration / time.Millisecond))
	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "rehash complete", "disconnectCount", disconnectCount, "duration", duration)
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
		r.connector.DisconnectAll(device.CloseReason{Err: e.Err, Text: ServiceDiscoveryError})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryError).Add(1.0)

	case e.Stopped:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery monitor being stopped")
		r.connector.DisconnectAll(device.CloseReason{Text: ServiceDiscoveryStopped})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryStopped).Add(1.0)

	case e.EventCount == 1:
		logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "ignoring initial instances")

	case len(e.Instances) > 0:
		r.rehash(e.Service, logger, r.accessorFactory(e.Instances))

	default:
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "disconnecting all devices: service discovery updated with no instances")
		r.connector.DisconnectAll(device.CloseReason{Text: ServiceDiscoveryNoInstances})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances).Add(1.0)
	}
}
