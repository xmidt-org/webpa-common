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
