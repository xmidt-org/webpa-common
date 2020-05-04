package monitor

import (
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/go-kit/kit/sd"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/service"
)

// Event carries the same information as go-kit's sd.Event, but with the extra Key that identifies
// which filtered service or path was updated.
type Event struct {
	// Key is the in-process unique identifier for the sd.Instancer that produced this event.
	// For consul, this value does not equal the name of the service (as it includes tags, datacenters, etc.).
	// For that purpose, use the Service field.
	Key string

	// Service, unlike Key, specifically identifies the service of the sd.Instancer that produced this event.
	// This value is used by listeners to update metric labels.
	Service string

	// Instancer is the go-kit sd.Instancer which sent this event.  This instance can be used to enrich
	// logging via logging.Enrich.
	Instancer sd.Instancer

	// EventCount is the postive, ascending integer identifying this event's sequence, e.g. 1 refers to the first
	// service discovery event.  Useful for logging and certain types of logic, such as ignoring the initial instances from a monitor.
	EventCount int

	// Instances are the filtered instances that came from the sd.Instancer.  If this is set,
	// Err will be nil.
	Instances []string

	// Err is any service discovery error that occurred.  If this is set, Instances will be empty.
	Err error

	// Stopped is set to true if and only if this event is being sent to indicate the monitoring goroutine
	// has exited, either because of being explicitly stopped or because the environment was closed.
	Stopped bool
}

type Listener interface {
	MonitorEvent(Event)
}

type ListenerFunc func(Event)

func (lf ListenerFunc) MonitorEvent(e Event) {
	lf(e)
}

type Listeners []Listener

func (ls Listeners) MonitorEvent(e Event) {
	for _, v := range ls {
		v.MonitorEvent(e)
	}
}

// NewMetricsListener produces a monitor Listener that gathers metrics related to service discovery.
func NewMetricsListener(p provider.Provider) Listener {
	var (
		errorCount    = p.NewCounter(service.ErrorCount)
		lastError     = p.NewGauge(service.LastErrorTimestamp)
		updateCount   = p.NewCounter(service.UpdateCount)
		lastUpdate    = p.NewGauge(service.LastUpdateTimestamp)
		instanceCount = p.NewGauge(service.InstanceCount)
	)

	return ListenerFunc(func(e Event) {
		timestamp := float64(time.Now().Unix())

		if e.Err != nil {
			errorCount.With(service.ServiceLabel, e.Service).Add(1.0)
			lastError.With(service.ServiceLabel, e.Service).Set(timestamp)
		} else {
			updateCount.With(service.ServiceLabel, e.Service).Add(1.0)
			lastUpdate.With(service.ServiceLabel, e.Service).Set(timestamp)
		}

		instanceCount.With(service.ServiceLabel, e.Service).Set(float64(len(e.Instances)))
	})
}

// NewAccessorListener creates a service discovery Listener that dispatches accessor instances to a nested closure.
// Any error received from the event results in a nil Accessor together with that error being passed to the next closure.
// If the AccessorFactory is nil, DefaultAccessorFactory is used.  If the next closure is nil, this function panics.
//
// An UpdatableAccessor may directly be used to receive events by passing Update as the next closure:
//    ua := new(UpdatableAccessor)
//    l := NewAccessorListener(f, ua.Update)
func NewAccessorListener(f service.AccessorFactory, next func(service.Accessor, error)) Listener {
	if next == nil {
		panic("A next closure is required to receive Accessors")
	}

	if f == nil {
		f = service.DefaultAccessorFactory
	}

	return ListenerFunc(func(e Event) {
		switch {
		case e.Err != nil:
			next(nil, e.Err)

		case len(e.Instances) > 0:
			next(f(e.Instances), nil)

		default:
			next(service.EmptyAccessor(), nil)
		}
	})
}

func NewKeyAccessorListener(f service.AccessorFactory, key string, next func(string, service.Accessor, error)) Listener {
	if next == nil {
		panic("A next closure is required to receive Accessors")
	}

	if f == nil {
		f = service.DefaultAccessorFactory
	}

	return ListenerFunc(func(e Event) {
		switch {
		case e.Err != nil:
			next(key, nil, e.Err)

		case len(e.Instances) > 0:
			next(key, f(e.Instances), nil)

		default:
			next(key, service.EmptyAccessor(), nil)
		}
	})
}

const (
	stateDeregistered uint32 = 0
	stateRegistered   uint32 = 1
)

// NewRegistrarListener binds service registration to the lifecycle of a service discovery watch.
// Upon the first successful update, or on any successful update following one or more errors, the given
// registrar is registered.  Any error that follows a successful update, or on the first error, results
// in deregistration.
func NewRegistrarListener(logger log.Logger, r sd.Registrar, initiallyRegistered bool) Listener {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	if r == nil {
		panic("A registrar is required")
	}

	var state uint32 = stateDeregistered
	if initiallyRegistered {
		state = stateRegistered
	}

	return ListenerFunc(func(e Event) {
		var message string
		if e.Err != nil {
			message = "deregistering on service discovery error"
		} else if e.Stopped {
			message = "deregistering due to monitor being stopped"
		} else {
			if atomic.CompareAndSwapUint32(&state, stateDeregistered, stateRegistered) {
				logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "registering on service discovery update")
				r.Register()
			}

			return
		}

		if atomic.CompareAndSwapUint32(&state, stateRegistered, stateDeregistered) {
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), message, logging.ErrorKey(), e.Err, "stopped", e.Stopped)
			r.Deregister()
		}
	})
}
