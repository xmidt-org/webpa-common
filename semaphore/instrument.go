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

// WithFailures establishes a metric that tracks how many times a resource was unable to
// be acquired, due to timeouts, context cancellations, etc.
func WithFailures(a xmetrics.Adder) InstrumentOption {
	return func(i *instrumentedSemaphore) {
		if a != nil {
			i.failures = a
		} else {
			i.failures = discard.NewCounter()
		}
	}
}

// Instrument decorates an existing semaphore with a set of options.
func Instrument(s Interface, o ...InstrumentOption) Interface {
	if s == nil {
		panic("A delegate semaphore is required")
	}

	is := &instrumentedSemaphore{
		Interface: s,
		resources: discard.NewCounter(),
		failures:  discard.NewCounter(),
	}

	for _, f := range o {
		f(is)
	}

	return is
}

type instrumentedSemaphore struct {
	Interface
	resources xmetrics.Adder
	failures  xmetrics.Adder
}

func (is *instrumentedSemaphore) Acquire() {
	is.Interface.Acquire()
	is.resources.Add(1.0)
}

func (is *instrumentedSemaphore) AcquireWait(t <-chan time.Time) (err error) {
	err = is.Interface.AcquireWait(t)
	if err != nil {
		is.failures.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) AcquireCtx(ctx context.Context) (err error) {
	err = is.Interface.AcquireCtx(ctx)
	if err != nil {
		is.failures.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) TryAcquire() (acquired bool) {
	acquired = is.Interface.TryAcquire()
	if acquired {
		is.resources.Add(1.0)
	} else {
		is.failures.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) Release() {
	is.Interface.Release()
	is.resources.Add(-1.0)
}
