package monitor

import "github.com/Comcast/webpa-common/service"

// Event carries the same information as go-kit's sd.Event, but with the extra Key that identifies
// which service key or path was updated.
type Event struct {
	// Key is the in-process identifier for the sd.Instancer that produced this event
	Key string

	// Instances are the filtered instances that came from the sd.Instancer.  If this is set,
	// Err will be nil.
	Instances []string

	// Err is any service discovery error that occurred.  If this is set, Instances will be empty.
	Err error

	// Stopped is set to true if and only if this event is being sent to indicate the monitoring goroutine
	// has exited, either because of being explicitly stopped or because the environment was closed.
	Stopped bool
}

type Listener func(Event)

type Listeners []Listener

func (ls Listeners) Dispatch(e Event) {
	for _, l := range ls {
		l(e)
	}
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

	return func(e Event) {
		switch {
		case e.Err != nil:
			next(nil, e.Err)

		case len(e.Instances) > 0:
			next(f(e.Instances), nil)

		default:
			next(service.EmptyAccessor(), nil)
		}
	}
}
