package tracing

import (
	"sync"
	"time"
)

// Spanner acts as a factory for Spans
type Spanner interface {
	// Start begins a new, unfinished span.  The returned closure must be called
	// to finished the span, recording it with a duration and the given error.  The
	// returned closure is idempotent and only records the duration and error of the first call.
	// It always returns the same Span instance.
	Start(string) func(error) Span
}

func NewSpanner() Spanner {
	return &spanner{
		now:   time.Now,
		since: time.Since,
	}
}

type spanner struct {
	lock  sync.RWMutex
	now   func() time.Time
	since func(time.Time) time.Duration
}

func (sp *spanner) Start(name string) func(error) Span {
	s := &span{
		name:  name,
		start: sp.now(),
	}

	return func(err error) Span {
		s.finish(sp.since(s.start), err)
		return s
	}
}
