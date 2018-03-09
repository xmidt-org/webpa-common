package service

// Event carries the same information as go-kit's sd.Event, but with the extra Key that identifies
// which service key or path was updated.
type Event struct {
	Key       string
	Instances []string
	Err       error
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
func NewAccessorListener(f AccessorFactory, next func(Accessor, error)) Listener {
	if next == nil {
		panic("A next closure is required to receive Accessors")
	}

	if f == nil {
		f = DefaultAccessorFactory
	}

	return func(e Event) {
		switch {
		case e.Err != nil:
			next(nil, e.Err)

		case len(e.Instances) > 0:
			next(f(e.Instances), nil)

		default:
			next(EmptyAccessor(), nil)
		}
	}
}
