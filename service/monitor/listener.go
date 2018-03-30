package monitor

import (
	"sync"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/go-kit/kit/sd"
)

// Event carries the same information as go-kit's sd.Event, but with the extra Key that identifies
// which service key or path was updated.
type Event struct {
	// Key is the in-process identifier for the sd.Instancer that produced this event
	Key string

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
			errorCount.With(service.ServiceLabel, e.Key).Add(1.0)
			lastError.With(service.ServiceLabel, e.Key).Set(timestamp)
		} else {
			updateCount.With(service.ServiceLabel, e.Key).Add(1.0)
			lastUpdate.With(service.ServiceLabel, e.Key).Set(timestamp)
		}

		instanceCount.With(service.ServiceLabel, e.Key).Set(float64(len(e.Instances)))
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

// NewRegistrarListener binds service registration to the lifecycle of a service discovery watch.
// Upon the first successful update, or on any successful update following one or more errors, the given
// registrar is registered.  Any error that follows a successful update, or on the first error, results
// in deregistration.
func NewRegistrarListener(logger log.Logger, r sd.Registrar) Listener {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	if r == nil {
		panic("A registrar is required")
	}

	var (
		// this listener can be called from multiple goroutines if more than one watch has been set
		lock         sync.Mutex
		hasSucceeded bool
		errorCount   int
	)

	return ListenerFunc(func(e Event) {
		defer lock.Unlock()
		lock.Lock()

		if len(e.Instances) > 0 && e.Err == nil {
			if !hasSucceeded || errorCount > 0 {
				logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "registering due to service discovery update")
				r.Register()
			}

			hasSucceeded = true
			errorCount = 0
			return
		} else if errorCount == 0 {
			// if the first event is an error, this case will execute.
			// this shouldn't be an issue, as Deregister is idempotent.
			logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "deregistering due to service discovery error", logging.ErrorKey(), e.Err, "instanceCount", len(e.Instances))
			r.Deregister()
		}

		errorCount++
	})
}
