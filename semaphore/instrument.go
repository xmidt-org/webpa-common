package semaphore

import (
	"context"
	"time"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/discard"
)

// InstrumentOption represents a configurable option for instrumenting a semaphore
type InstrumentOption func(*instrumentedSemaphore)

// WithResources establishes a metric that tracks the resource count of the semaphore.
// If a nil counter is supplied, resource counts are discarded.
func WithResources(a xmetrics.Adder) InstrumentOption {
	return func(i *instrumentedSemaphore) {
		if a != nil {
			i.resources = a
		} else {
			i.resources = discard.NewCounter()
		}
	}
}

// WithErrors establishes a metric that tracks how many errors, or failed resource acquisitions,
// happen when attempting to acquire resources.  If a nil counter is supplied, error counts
// are discarded.
func WithErrors(a xmetrics.Adder) InstrumentOption {
	return func(i *instrumentedSemaphore) {
		if a != nil {
			i.errors = a
		} else {
			i.errors = discard.NewCounter()
		}
	}
}

// Instrument decorates an existing semaphore with a set of options.
func Instrument(s Interface, o ...InstrumentOption) Interface {
	is := &instrumentedSemaphore{
		Interface: s,
		resources: discard.NewCounter(),
		errors:    discard.NewCounter(),
	}

	for _, f := range o {
		f(is)
	}

	return is
}

type instrumentedSemaphore struct {
	Interface
	resources xmetrics.Adder
	errors    xmetrics.Adder
}

func (is *instrumentedSemaphore) Acquire() {
	is.Interface.Acquire()
	is.resources.Add(1.0)
}

func (is *instrumentedSemaphore) AcquireWait(t <-chan time.Time) (err error) {
	err = is.Interface.AcquireWait(t)
	if err != nil {
		is.errors.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) AcquireCtx(ctx context.Context) (err error) {
	err = is.Interface.AcquireCtx(ctx)
	if err != nil {
		is.errors.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) Release() {
	is.Interface.Release()
	is.resources.Add(-1.0)
}
