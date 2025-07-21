// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package semaphore

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithResources(t *testing.T) {
	var (
		assert = assert.New(t)
		io     = new(instrumentOptions)

		custom = generic.NewCounter("test")
	)

	WithResources(nil)(io)
	assert.NotNil(io.resources)

	WithResources(custom)(io)
	assert.Equal(custom, io.resources)
}

func TestWithFailures(t *testing.T) {
	var (
		assert = assert.New(t)
		io     = new(instrumentOptions)

		custom = generic.NewCounter("test")
	)

	WithFailures(nil)(io)
	assert.NotNil(io.failures)

	WithFailures(custom)(io)
	assert.Equal(custom, io.failures)
}

func TestWithClosed(t *testing.T) {
	var (
		assert = assert.New(t)
		io     = new(instrumentOptions)

		custom = generic.NewGauge("test")
	)

	WithClosed(nil)(io)
	assert.NotNil(io.closed)

	WithClosed(custom)(io)
	assert.Equal(custom, io.closed)
}

func testInstrumentNilSemaphore(t *testing.T) {
	assert.Panics(t,
		func() {
			Instrument(nil)
		},
	)
}

func TestInstrument(t *testing.T) {
	t.Run("NilSemaphore", testInstrumentNilSemaphore)
}

func testInstrumentCloseableNilSemaphore(t *testing.T) {
	assert.Panics(t,
		func() {
			InstrumentCloseable(nil)
		},
	)
}

func TestInstrumentCloseable(t *testing.T) {
	t.Run("NilSemaphore", testInstrumentCloseableNilSemaphore)
}

func testInstrumentedSemaphoreAcquireSuccess(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))

		result = make(chan error)
	)

	go func() {
		result <- s.Acquire()
	}()

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())

		assert.NoError(s.Release())
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("Acquire blocked unexpectedly")
	}
}

func testInstrumentedSemaphoreAcquireFail(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		cm        = CloseableMutex()
		s         = Instrument(cm, WithResources(resources), WithFailures(failures))

		result = make(chan error)
	)

	go func() {
		// nolint: typecheck
		cm.Close()
		result <- s.Acquire()
	}()

	select {
	case err := <-result:
		assert.Equal(ErrClosed, err)
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())

		assert.Equal(ErrClosed, s.Release()) // idempotent
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("Acquire blocked unexpectedly")
	}
}

func testInstrumentedSemaphoreTryAcquire(t *testing.T) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))
	)

	assert.Zero(resources.Value())
	assert.Zero(failures.Value())

	require.True(s.TryAcquire())
	assert.Equal(float64(1.0), resources.Value())
	assert.Zero(failures.Value())

	require.False(s.TryAcquire())
	assert.Equal(float64(1.0), resources.Value())
	assert.Equal(float64(1.0), failures.Value())

	assert.NoError(s.Release())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
}

func testInstrumentedSemaphoreAcquireWaitSuccess(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))

		ready  = make(chan struct{})
		result = make(chan error)
		timer  = make(chan time.Time)
	)

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		s.Release()
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireWait goroutine")
	}

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())

		assert.NoError(s.Release())
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireWait blocked unexpectedly")
	}
}

func testInstrumentedSemaphoreAcquireWaitTimeout(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))

		ready  = make(chan struct{})
		result = make(chan error)
		timer  = make(chan time.Time)
	)

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		timer <- time.Time{}
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireWait goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ErrTimeout, err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireWait blocked unexpectedly")
	}
}

func testInstrumentedSemaphoreAcquireCtxSuccess(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))

		ready       = make(chan struct{})
		result      = make(chan error)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		s.Release()
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireCtx goroutine")
	}

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireCtx blocked unexpectedly")
	}
}

func testInstrumentedSemaphoreAcquireCtxCancel(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		s         = Instrument(Mutex(), WithResources(resources), WithFailures(failures))

		ready       = make(chan struct{})
		result      = make(chan error)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		cancel()
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireCtx goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ctx.Err(), err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireCtx blocked unexpectedly")
	}
}

func TestInstrumentedSemaphore(t *testing.T) {
	t.Run("Acquire", func(t *testing.T) {
		t.Run("Success", testInstrumentedSemaphoreAcquireSuccess)
		t.Run("Fail", testInstrumentedSemaphoreAcquireFail)
	})

	t.Run("TryAcquire", testInstrumentedSemaphoreTryAcquire)

	t.Run("AcquireWait", func(t *testing.T) {
		t.Run("Success", testInstrumentedSemaphoreAcquireWaitSuccess)
		t.Run("Timeout", testInstrumentedSemaphoreAcquireWaitTimeout)
	})

	t.Run("AcquireCtx", func(t *testing.T) {
		t.Run("Success", testInstrumentedSemaphoreAcquireCtxSuccess)
		t.Run("Cancel", testInstrumentedSemaphoreAcquireCtxCancel)
	})
}

func testInstrumentedCloseableAcquire(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		result = make(chan error)
	)

	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		result <- s.Acquire()
	}()

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		assert.NoError(s.Release())
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("Acquire blocked unexpectedly")
	}

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Zero(failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	assert.Equal(ErrClosed, s.Acquire())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func testInstrumentedCloseableTryAcquire(t *testing.T) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))
	)

	assert.Zero(resources.Value())
	assert.Zero(failures.Value())
	assert.Equal(MetricOpen, closed.Value())

	require.True(s.TryAcquire())
	assert.Equal(float64(1.0), resources.Value())
	assert.Zero(failures.Value())
	assert.Equal(MetricOpen, closed.Value())

	require.False(s.TryAcquire())
	assert.Equal(float64(1.0), resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricOpen, closed.Value())

	assert.NoError(s.Release())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricOpen, closed.Value())

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	assert.False(s.TryAcquire())
	assert.Zero(resources.Value())
	assert.Equal(float64(2.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func testInstrumentedCloseableAcquireWaitSuccess(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready  = make(chan struct{})
		result = make(chan error)
		timer  = make(chan time.Time)
	)

	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
		s.Release()
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireWait goroutine")
	}

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		assert.NoError(s.Release())
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireWait blocked unexpectedly")
	}

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Zero(failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func testInstrumentedCloseableAcquireWaitTimeout(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready  = make(chan struct{})
		result = make(chan error)
		timer  = make(chan time.Time)
	)

	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
		timer <- time.Time{}
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireWait goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ErrTimeout, err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireWait blocked unexpectedly")
	}

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func testInstrumentedCloseableAcquireWaitClose(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready  = make(chan struct{})
		result = make(chan error)
		timer  = make(chan time.Time)
	)

	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireWait(timer)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
		// nolint: typecheck
		assert.NoError(s.Close())
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireWait goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ErrClosed, err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricClosed, closed.Value())

		s.Release()
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricClosed, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireWait blocked unexpectedly")
	}

	// nolint: typecheck
	assert.Equal(ErrClosed, s.Close())
	assert.Equal(float64(1.0), resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}

	assert.Equal(ErrClosed, s.AcquireWait(timer))
	assert.Equal(float64(1.0), resources.Value())
	assert.Equal(float64(2.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())
}

func testInstrumentedCloseableAcquireCtxSuccess(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready       = make(chan struct{})
		result      = make(chan error)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()
	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
		s.Release()

		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireCtx goroutine")
	}

	select {
	case err := <-result:
		assert.NoError(err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireCtx blocked unexpectedly")
	}

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Zero(failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}

	assert.Equal(ErrClosed, s.AcquireCtx(ctx))
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())
}

func testInstrumentedCloseableAcquireCtxCancel(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready       = make(chan struct{})
		result      = make(chan error)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()
	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())
		cancel()
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireCtx goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ctx.Err(), err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		s.Release()
		assert.Zero(resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricOpen, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireCtx blocked unexpectedly")
	}

	// nolint: typecheck
	assert.NoError(s.Close())
	assert.Zero(resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func testInstrumentedCloseableAcquireCtxClose(t *testing.T) {
	var (
		assert    = assert.New(t)
		resources = generic.NewCounter("test")
		failures  = generic.NewCounter("test")
		closed    = generic.NewGauge("test")
		s         = InstrumentCloseable(CloseableMutex(), WithResources(resources), WithFailures(failures), WithClosed(closed))

		ready       = make(chan struct{})
		result      = make(chan error)
		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()
	// nolint: typecheck
	assert.NotNil(s.Closed())
	assert.Equal(MetricOpen, closed.Value())

	go func() {
		s.Acquire()
		close(ready)
		result <- s.AcquireCtx(ctx)
	}()

	select {
	case <-ready:
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricOpen, closed.Value())

		// nolint: typecheck
		assert.NoError(s.Close())
		assert.Equal(float64(1.0), resources.Value())
		assert.Zero(failures.Value())
		assert.Equal(MetricClosed, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("Failed to spawn AcquireCtx goroutine")
	}

	select {
	case err := <-result:
		assert.Equal(ErrClosed, err)
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricClosed, closed.Value())

		assert.Equal(ErrClosed, s.Release())
		assert.Equal(float64(1.0), resources.Value())
		assert.Equal(float64(1.0), failures.Value())
		assert.Equal(MetricClosed, closed.Value())
	case <-time.After(time.Second):
		assert.FailNow("AcquireCtx blocked unexpectedly")
	}

	// nolint: typecheck
	assert.Equal(ErrClosed, s.Close())
	assert.Equal(float64(1.0), resources.Value())
	assert.Equal(float64(1.0), failures.Value())
	assert.Equal(MetricClosed, closed.Value())

	select {
	// nolint: typecheck
	case <-s.Closed():
		// passing
	default:
		assert.Fail("The Closed channel was not signaled")
	}
}

func TestInstrumentedCloseable(t *testing.T) {
	t.Run("Acquire", testInstrumentedCloseableAcquire)

	t.Run("TryAcquire", testInstrumentedCloseableTryAcquire)

	t.Run("AcquireWait", func(t *testing.T) {
		t.Run("Success", testInstrumentedCloseableAcquireWaitSuccess)
		t.Run("Timeout", testInstrumentedCloseableAcquireWaitTimeout)
		t.Run("Close", testInstrumentedCloseableAcquireWaitClose)
	})

	t.Run("AcquireCtx", func(t *testing.T) {
		t.Run("Success", testInstrumentedCloseableAcquireCtxSuccess)
		t.Run("Cancel", testInstrumentedCloseableAcquireCtxCancel)
		t.Run("Close", testInstrumentedCloseableAcquireCtxClose)
	})
}
