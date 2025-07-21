// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package semaphore

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics/discard"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const (
	MetricOpen   float64 = 1.0
	MetricClosed float64 = 0.0
)

type instrumentOptions struct {
	failures  xmetrics.Adder
	resources xmetrics.Adder
	closed    xmetrics.Setter
}

var defaultOptions = instrumentOptions{
	failures:  discard.NewCounter(),
	resources: discard.NewCounter(),
	closed:    discard.NewGauge(),
}

// InstrumentOption represents a configurable option for instrumenting a semaphore
type InstrumentOption func(*instrumentOptions)

// WithResources establishes a metric that tracks the resource count of the semaphore.
// If a nil counter is supplied, resource counts are discarded.
func WithResources(a xmetrics.Adder) InstrumentOption {
	return func(io *instrumentOptions) {
		if a != nil {
			io.resources = a
		} else {
			io.resources = discard.NewCounter()
		}
	}
}

// WithFailures establishes a metric that tracks how many times a resource was unable to
// be acquired, due to timeouts, context cancellations, etc.
func WithFailures(a xmetrics.Adder) InstrumentOption {
	return func(io *instrumentOptions) {
		if a != nil {
			io.failures = a
		} else {
			io.failures = discard.NewCounter()
		}
	}
}

// WithClosed sets a gauge that records the state of a Closeable semaphore, 1.0 for open and 0.0 for closed.
// This option is ignored for regular semaphores.
func WithClosed(s xmetrics.Setter) InstrumentOption {
	return func(io *instrumentOptions) {
		if s != nil {
			io.closed = s
		} else {
			io.closed = discard.NewGauge()
		}
	}
}

// Instrument decorates an existing semaphore with instrumentation.  The available options
// allow tracking the number of resources currently acquired and the total count of failures over time.
// The returned Interface object will not implement Closeable, even if the decorated semaphore does.
func Instrument(s Interface, o ...InstrumentOption) Interface {
	if s == nil {
		panic("A delegate semaphore is required")
	}

	io := defaultOptions

	for _, f := range o {
		f(&io)
	}

	return &instrumentedSemaphore{
		delegate:  s,
		failures:  io.failures,
		resources: io.resources,
	}
}

// InstrumentCloseable is similar to Instrument, but works with Closeable semaphores.  The WithClosed
// option is honored by this factory function.
func InstrumentCloseable(c Closeable, o ...InstrumentOption) Closeable {
	if c == nil {
		panic("A delegate semaphore is required")
	}

	io := defaultOptions

	for _, f := range o {
		f(&io)
	}

	ic := &instrumentedCloseable{
		instrumentedSemaphore: instrumentedSemaphore{
			delegate:  c,
			failures:  io.failures,
			resources: io.resources,
		},
		closed: io.closed,
	}

	ic.closed.Set(MetricOpen)
	return ic
}

// instrumentedSemaphore is the internal decorator around Interface that applies appropriate metrics.
type instrumentedSemaphore struct {
	delegate  Interface
	resources xmetrics.Adder
	failures  xmetrics.Adder
}

func (is *instrumentedSemaphore) Acquire() (err error) {
	err = is.delegate.Acquire()
	if err != nil {
		is.failures.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) AcquireWait(t <-chan time.Time) (err error) {
	err = is.delegate.AcquireWait(t)
	if err != nil {
		is.failures.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) AcquireCtx(ctx context.Context) (err error) {
	err = is.delegate.AcquireCtx(ctx)
	if err != nil {
		is.failures.Add(1.0)
	} else {
		is.resources.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) TryAcquire() (acquired bool) {
	acquired = is.delegate.TryAcquire()
	if acquired {
		is.resources.Add(1.0)
	} else {
		is.failures.Add(1.0)
	}

	return
}

func (is *instrumentedSemaphore) Release() (err error) {
	err = is.delegate.Release()
	if err == nil {
		is.resources.Add(-1.0)
	}

	return
}

type instrumentedCloseable struct {
	instrumentedSemaphore
	closed xmetrics.Setter
}

func (ic *instrumentedCloseable) Close() (err error) {
	// nolint: typecheck
	err = (ic.instrumentedSemaphore.delegate).(Closeable).Close()
	ic.closed.Set(MetricClosed)

	// NOTE: we don't set the resources metric to 0 as a way of preserving the state
	// for debugging.  Can change this if desired.

	return
}

func (ic *instrumentedCloseable) Closed() <-chan struct{} {
	return (ic.instrumentedSemaphore.delegate).(Closeable).Closed()
}
