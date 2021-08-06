package monitor

import (
	"errors"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/service"
)

var errNoInstances = errors.New("No instances to monitor")

// Interface represents an active monitor for one or more sd.Instancer objects.
type Interface interface {
	// Stopped returns a channel that is closed when this Monitor is stopped.
	// Semantics are equivalent to context.Context.Done().
	Stopped() <-chan struct{}

	// Stop halts all goroutines that are dispatching events, but does not stop
	// or close the service discovery backend.  This method is idempotent.  Once stopped,
	// a Monitor cannot be reused.
	Stop()
}

// Option represents a configuration option for a monitor
type Option func(*monitor)

// WithLogger sets a go-kit Logger for this monitor.  This logger will be enriched with information
// about each instancer, if available.  If nil, the default logger is used instead.
func WithLogger(l log.Logger) Option {
	return func(m *monitor) {
		if l == nil {
			m.logger = logging.DefaultLogger()
		} else {
			m.logger = l
		}
	}
}

// WithClosed sets an external channel that, when closed, will cause all goroutines spawned
// by this monitor to exit.  This is useful when orchestrating multiple monitors, or when restarting
// service discovery clients.
func WithClosed(c <-chan struct{}) Option {
	return func(m *monitor) {
		m.closed = c
	}
}

// WithFilter establishes the filtering strategy for discovered service instances.  By default, TrimAndSortFilter is used.
// If the filter is nil, filtering is disabled and every Listener will receive the raw, unfiltered instances from the
// service discovery backend.
func WithFilter(f Filter) Option {
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
func WithListeners(l ...Listener) Option {
	return func(m *monitor) {
		if len(l) > 0 {
			m.listeners = append(Listeners{}, l...)
		} else {
			m.listeners = nil
		}
	}
}

// WithInstancers establishes the set of sd.Instancer objects to be monitored.  The given Instancers
// is copied to maintain the monitor's immutability.
func WithInstancers(i service.Instancers) Option {
	return func(m *monitor) {
		m.instancers = i.Copy()
	}
}

func WithEnvironment(e service.Environment) Option {
	return func(m *monitor) {
		m.instancers = e.Instancers()
		m.closed = e.Closed()
	}
}

// New begins monitoring one or more sd.Instancer objects, dispatching events to any Listeners that are configured.
// This function returns an error if i is empty or nil.
func New(options ...Option) (Interface, error) {
	var (
		m = &monitor{
			logger:  logging.DefaultLogger(),
			stopped: make(chan struct{}),
			filter:  DefaultFilter(),
		}
	)

	for _, o := range options {
		o(m)
	}

	if err := m.start(); err != nil {
		return nil, err
	}

	return m, nil
}

// monitor is the internal implementation of Monitor.  This type is a shared context
// among all goroutines that monitor a (key, instancer) pair.
type monitor struct {
	logger     log.Logger
	instancers service.Instancers
	filter     Filter
	listeners  Listeners

	closed   <-chan struct{}
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

func (m *monitor) start() error {
	if m.instancers.Len() == 0 {
		return errNoInstances
	}

	for k, v := range m.instancers {
		var svc = k
		if ci, ok := v.(service.ContextualInstancer); ok {
			if svcName, ok := ci.Metadata()["service"].(string); ok {
				svc = svcName
			}
		}
		go m.dispatchEvents(k, svc, logging.Enrich(m.logger, v), v)
	}

	return nil
}

// dispatchEvents is a goroutine that consumes service discovery events from an sd.Instancer
// and dispatches those events zero or more Listeners.  If configured, the filter is used to
// preprocess the set of instances sent to the listener.
func (m *monitor) dispatchEvents(key, service string, l log.Logger, i sd.Instancer) {
	var (
		eventCount              = 0
		eventCounter log.Valuer = func() interface{} {
			return eventCount
		}

		logger = log.With(l, EventCountKey(), eventCounter)
		events = make(chan sd.Event, 10)
	)

	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "subscription monitor starting")

	defer i.Deregister(events)
	i.Register(events)

	for {
		select {
		case sdEvent := <-events:
			eventCount++
			event := Event{
				Key:        key,
				Service:    service,
				Instancer:  i,
				EventCount: eventCount,
			}

			if sdEvent.Err != nil {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "service discovery error", logging.ErrorKey(), sdEvent.Err)
				event.Err = sdEvent.Err
			} else {
				logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "service discovery update", "instances", sdEvent.Instances)
				if len(sdEvent.Instances) > 0 {
					event.Instances = m.filter(sdEvent.Instances)
				}
			}

			m.listeners.MonitorEvent(event)

		case <-m.stopped:
			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "subscription monitor was stopped")
			m.listeners.MonitorEvent(Event{Key: key, Service: service, Instancer: i, EventCount: eventCount, Stopped: true})
			return

		case <-m.closed:
			logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "subscription monitor exiting due to external closure")
			m.Stop() // ensure that the Stopped state is correct
			m.listeners.MonitorEvent(Event{Key: key, Service: service, Instancer: i, EventCount: eventCount, Stopped: true})
			return
		}
	}
}
