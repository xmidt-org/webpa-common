package rehasher

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"go.uber.org/zap"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/device"
	"github.com/xmidt-org/webpa-common/v2/service"
	"github.com/xmidt-org/webpa-common/v2/service/monitor"
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
func WithLogger(l *zap.Logger) Option {
	return func(r *rehasher) {
		if l == nil {
			r.logger = sallust.Default()
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

// New creates a monitor Listener which will rehash and disconnect devices in response to service discovery events
// from a given set of services.
// This function panics if the connector is nil, if no IsRegistered strategy is configured or if no services were
// provided to filter events.
//
// If the returned listener encounters any service discovery error, all devices are disconnected.  Otherwise,
// the IsRegistered strategy is used to determine which devices should still be connected to the Connector.  Devices
// that hash to instances not registered in this environment are disconnected.
func New(connector device.Connector, services []string, options ...Option) monitor.Listener {
	if connector == nil {
		panic("A device Connector is required.")
	}

	if len(services) < 1 {
		panic("Services are required to avoid unintended reshashes.")
	}

	var (
		defaultProvider = provider.NewDiscardProvider()

		r = &rehasher{
			logger:          sallust.Default(),
			accessorFactory: service.DefaultAccessorFactory,
			connector:       connector,
			now:             time.Now,
			services:        make(map[string]bool),

			keep:                 defaultProvider.NewGauge(RehashKeepDevice),
			disconnect:           defaultProvider.NewGauge(RehashDisconnectDevice),
			disconnectAllCounter: defaultProvider.NewCounter(RehashDisconnectAllCounter),
			timestamp:            defaultProvider.NewGauge(RehashTimestamp),
			duration:             defaultProvider.NewGauge(RehashDurationMilliseconds),
		}
	)

	for _, svc := range services {
		r.services[svc] = true
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
	logger          *zap.Logger
	services        map[string]bool
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

func (r *rehasher) rehash(svc string, logger *zap.Logger, accessor service.Accessor) {
	logger.Info("rehash starting")

	start := r.now()
	r.timestamp.With(service.ServiceLabel, svc).Set(float64(start.UTC().Unix()))

	var (
		keepCount = 0

		disconnectCount = r.connector.DisconnectIf(func(candidate device.ID) (device.CloseReason, bool) {
			instance, err := accessor.Get(candidate.Bytes())
			switch {
			case err != nil:
				logger.Error("disconnecting device: error during rehash",
					zap.Error(err),
					zap.String("id", string(candidate)),
				)

				return device.CloseReason{Err: err, Text: RehashError}, true

			case !r.isRegistered(instance):
				logger.Info("disconnecting device: rehashed to another instance",
					zap.String("instance", instance),
					zap.String("id", string(candidate)),
				)

				return device.CloseReason{Text: RehashOtherInstance}, true

			default:
				logger.Debug("device hashed to this instance", zap.String("id", string(candidate)))
				keepCount++
				return device.CloseReason{}, false
			}
		})

		duration = r.now().Sub(start)
	)

	r.keep.With(service.ServiceLabel, svc).Set(float64(keepCount))
	r.disconnect.With(service.ServiceLabel, svc).Set(float64(disconnectCount))
	r.duration.With(service.ServiceLabel, svc).Set(float64(duration / time.Millisecond))
	logger.Info("rehash complete", zap.Int("disconnectCount", disconnectCount), zap.Duration("duration", duration))
}

func (r *rehasher) MonitorEvent(e monitor.Event) {
	if !r.services[e.Service] {
		return
	}

	logger := sallust.Enrich(
		r.logger.With(
			zap.Int(monitor.EventCountKey(), e.EventCount),
		),
		e.Instancer,
	)

	switch {
	case e.Err != nil:
		logger.Error("disconnecting all devices: service discovery error", zap.Error(e.Err))
		r.connector.DisconnectAll(device.CloseReason{Err: e.Err, Text: ServiceDiscoveryError})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryError).Add(1.0)

	case e.Stopped:
		logger.Error("disconnecting all devices: service discovery monitor being stopped")
		r.connector.DisconnectAll(device.CloseReason{Text: ServiceDiscoveryStopped})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryStopped).Add(1.0)

	case e.EventCount == 1:
		logger.Info("ignoring initial instances")

	case len(e.Instances) > 0:
		r.rehash(e.Service, logger, r.accessorFactory(e.Instances))

	default:
		logger.Error("disconnecting all devices: service discovery updated with no instances")
		r.connector.DisconnectAll(device.CloseReason{Text: ServiceDiscoveryNoInstances})
		r.disconnectAllCounter.With(service.ServiceLabel, e.Service, ReasonLabel, DisconnectAllServiceDiscoveryNoInstances).Add(1.0)
	}
}
