package service

import (
	"errors"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/go-kit/kit/sd"
)

// Monitor represents an active monitor for one or more sd.Instancer objects.
type Monitor interface {
	// Stopped returns a channel that is closed when this Monitor is stopped.
	// Semantics are equivalent to context.Context.Done().
	Stopped() <-chan struct{}

	// Stop halts all goroutines that are dispatching events, but does not stop
	// or close the service discovery backend.  This method is idempotent.  Once stopped,
	// a Monitor cannot be reused.
	Stop()
}

// MonitorOption represents a configuration option for a Monitor
type MonitorOption func(*monitor)

// WithMetricsProvider uses a given provider to create the metrics used by a Monitor.  If the provider is nil,
// metrics are discarded.
func WithMetricsProvider(p provider.Provider) MonitorOption {
	return func(m *monitor) {
		if p == nil {
			p = provider.NewDiscardProvider()
		}

		m.errorCount = p.NewCounter(ErrorCount)
		m.lastError = p.NewGauge(LastErrorTimestamp)
		m.updateCount = p.NewCounter(UpdateCount)
		m.lastUpdate = p.NewGauge(LastUpdateTimestamp)
		m.instanceCount = p.NewGauge(InstanceCount)
	}
}

// WithFilter establishes the filtering strategy for discovered service instances.  By default, TrimAndSortFilter is used.
// If the filter is nil, filtering is disabled and every Listener will receive the raw, unfiltered instances from the
// service discovery backend.
func WithFilter(f Filter) MonitorOption {
	return func(m *monitor) {
		if f == nil {
			m.filter = NopFilter
		} else {
			m.filter = f
		}
	}
}

// WithListeners configures the monitor to dispatch to zero or more Listeners.  It is legal to start a Monitor
// with no Listeners, as this is equivalent to just logging messages for the service discovery backend.
func WithListeners(l ...Listener) MonitorOption {
	return func(m *monitor) {
		if len(l) > 0 {
			m.listeners = append(Listeners{}, l...)
		} else {
			m.listeners = nil
		}
	}
}

// WithNow establishes the closure used to fetch the system time.  By default, time.Now is used.  If passed nil,
// this option uses time.Now.
func WithNow(f func() time.Time) MonitorOption {
	return func(m *monitor) {
		if f == nil {
			m.now = time.Now
		} else {
			m.now = f
		}
	}
}

// StartMonitor begins monitoring one or more sd.Instancer objects, dispatching events to any Listeners that are configured.
// This function returns an error if i is empty or nil.
func StartMonitor(i Instancers, options ...MonitorOption) (Monitor, error) {
	if i.Len() == 0 {
		return nil, errors.New("No instancers to monitor")
	}

	var (
		defaultMetricsProvider = provider.NewDiscardProvider()

		m = &monitor{
			stopped: make(chan struct{}),
			filter:  DefaultFilter,
			now:     time.Now,

			errorCount:    defaultMetricsProvider.NewCounter(ErrorCount),
			lastError:     defaultMetricsProvider.NewGauge(LastErrorTimestamp),
			updateCount:   defaultMetricsProvider.NewCounter(UpdateCount),
			lastUpdate:    defaultMetricsProvider.NewGauge(LastUpdateTimestamp),
			instanceCount: defaultMetricsProvider.NewGauge(InstanceCount),
		}
	)

	for _, o := range options {
		o(m)
	}

	i.Each(func(k string, l log.Logger, v sd.Instancer) {
		go m.dispatchEvents(k, l, v)
	})

	return m, nil
}

// monitor is the internal implementation of Monitor.  This type is a shared context
// among all goroutines that monitor a (key, instancer) pair.
type monitor struct {
	filter    Filter
	listeners Listeners
	now       func() time.Time

	errorCount    metrics.Counter
	lastError     metrics.Gauge
	updateCount   metrics.Counter
	lastUpdate    metrics.Gauge
	instanceCount metrics.Gauge

	stopped  chan struct{}
	stopOnce sync.Once
}

func (m *monitor) Stopped() <-chan struct{} {
	return m.stopped
}

func (m *monitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopped)
	})
}

// timestamp is just a helper that returns the current Unix time as a float64
func (m *monitor) timestamp() float64 {
	return float64(m.now().Unix())
}

// dispatchEvents is a goroutine that consumes service discovery events from an sd.Instancer
// and dispatches those events zero or more Listeners.  If configured, the filter is used to
// preprocess the set of instances sent to the listener.
func (m *monitor) dispatchEvents(key string, l log.Logger, i sd.Instancer) {
	var (
		eventCount              = 0
		eventCounter log.Valuer = func() interface{} {
			return eventCount
		}

		logger = log.With(l, "eventCount", eventCounter)

		events = make(chan sd.Event, 10)

		errorCount = m.errorCount.With(ServiceLabel, key)
		lastError  = m.lastError.With(ServiceLabel, key)

		updateCount = m.updateCount.With(ServiceLabel, key)
		lastUpdate  = m.lastUpdate.With(ServiceLabel, key)

		instanceCount = m.instanceCount.With(ServiceLabel, key)
	)

	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "subscription monitor starting")

	defer i.Deregister(events)
	i.Register(events)

	for {
		select {
		case event := <-events:
			eventCount++

			if event.Err != nil {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "service discovery error", logging.ErrorKey(), event.Err)
				errorCount.Add(1.0)
				lastError.Set(m.timestamp())

				m.listeners.Dispatch(Event{Key: key, Err: event.Err})
			} else {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "service discovery update", "instances", event.Instances)
				updateCount.Add(1.0)
				lastUpdate.Set(m.timestamp())

				i := m.filter(event.Instances)
				instanceCount.Set(float64(len(i)))
				m.listeners.Dispatch(Event{Key: key, Instances: i})
			}

		case <-m.stopped:
			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "subscription monitor was shutdown")
			return
		}
	}
}
