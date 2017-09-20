package tracing

import (
	"time"
)

// Spanner acts as a factory for Spans
type Spanner interface {
	// Start begins a new, unfinished span.  The returned closure must be called
	// to finished the span, recording it with a duration and the given error.  The
	// returned closure is idempotent and only records the duration and error of the first call.
	// It always returns the same Span instance, and that instance is immutable once the
	// closure is called.
	Start(string) func(error) Span
}

// SpannerOption supplies a configuration option to a Spanner.
type SpannerOption func(*spanner)

// Now sets a now function on a spanner.  If now is nil, this option does nothing.
// This options is primarily useful for testing, however it can be useful in production
// situations.  For example, this option can be used to emit times with a consistent time zone, like UTC.
func Now(now func() time.Time) SpannerOption {
	return func(sp *spanner) {
		if now != nil {
			sp.now = now
		}
	}
}

// Since sets a since function on a spanner.  If since is nil, this option does nothing.
// This options is primarily useful for testing.
func Since(since func(time.Time) time.Duration) SpannerOption {
	return func(sp *spanner) {
		if since != nil {
			sp.since = since
		}
	}
}

// NewSpanner constructs a new Spanner with the given options.  By default, a Spanner
// will use time.Now() to get the current time and time.Since() to compute durations.
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

// spanner is the internal spanner implementation.
type spanner struct {
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
