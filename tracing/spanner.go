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

type SpannerOption func(*spanner)

// Now sets a now function on a spanner.  If now is nil, this option does nothing.
func Now(now func() time.Time) SpannerOption {
	return func(sp *spanner) {
		if now != nil {
			sp.now = now
		}
	}
}

// Since sets a since function on a spanner.  If since is nil, this option does nothing.
func Since(since func(time.Time) time.Duration) SpannerOption {
	return func(sp *spanner) {
		if since != nil {
			sp.since = since
		}
	}
}

// NewSpanner constructs a new Spanner with the given options
func NewSpanner(o ...SpannerOption) Spanner {
	sp := &spanner{
		now:   time.Now,
		since: time.Since,
	}

	for _, option := range o {
		option(sp)
	}

	return sp
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
