package clock

import "time"

// Timer represents an event source triggered at a particular time.  It is the analog of time.Timer.
type Timer interface {
	C() <-chan time.Time
	Reset(time.Duration) bool
	Stop() bool
}

type systemTimer struct {
	*time.Timer
}

func (st systemTimer) C() <-chan time.Time {
	return st.Timer.C
}

// WrapTimer wraps a time.Timer in a clock.Timer.  A typical usage would be
// WrapTimer(time.NewTimer(time.Second)).
func WrapTimer(t *time.Timer) Timer {
	return systemTimer{t}
}
