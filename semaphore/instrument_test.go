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
		is     = new(instrumentedSemaphore)

		custom = generic.NewCounter("test")
	)

	WithResources(nil)(is)
	assert.NotNil(is.resources)

	WithResources(custom)(is)
	assert.Equal(custom, is.resources)
}

func TestWithFailures(t *testing.T) {
	var (
		assert = assert.New(t)
		is     = new(instrumentedSemaphore)

		custom = generic.NewCounter("test")
	)

	WithFailures(nil)(is)
	assert.NotNil(is.failures)

	WithFailures(custom)(is)
	assert.Equal(custom, is.failures)
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

		s.Release()
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

		s.Release()
		assert.Zero(resources.Value())
		assert.Zero(failures.Value())
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

	s.Release()
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

		s.Release()
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
