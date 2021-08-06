package capacitor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/clock/clocktest"
)

func ExampleBasicUsage() {
	var (
		c = New()
		w = new(sync.WaitGroup)
	)

	w.Add(1)

	// this may or may not be executed, depending on timing of the machine where this is run
	c.Submit(func() {})

	// we'll wait until this is executed
	c.Submit(func() {
		fmt.Println("Discharged")
		w.Done()
	})

	w.Wait()

	// Output:
	// Discharged
}

func testWithDelayDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = new(capacitor)
	)

	WithDelay(0)(c)
	assert.Equal(DefaultDelay, c.delay)
}

func testWithDelayCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = new(capacitor)
	)

	WithDelay(31 * time.Minute)(c)
	assert.Equal(31*time.Minute, c.delay)
}

func TestWithDelay(t *testing.T) {
	t.Run("Default", testWithDelayDefault)
	t.Run("Custom", testWithDelayCustom)
}

func testWithClockDefault(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = new(capacitor)
	)

	WithClock(nil)(c)
	assert.NotNil(c.c)
}

func testWithClockCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		cl     = new(clocktest.Mock)
		c      = new(capacitor)
	)

	WithClock(cl)(c)
	assert.Equal(cl, c.c)
}

func TestWithClock(t *testing.T) {
	t.Run("Default", testWithClockDefault)
	t.Run("Custom", testWithClockCustom)
}

func testCapacitorSubmit(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		stopped = make(chan struct{})
		calls   int32
		f       = func() {
			atomic.AddInt32(&calls, 1)
		}

		cl      = new(clocktest.Mock)
		timer   = new(clocktest.MockTimer)
		trigger = make(chan time.Time, 1)
		c       = New(WithDelay(time.Minute), WithClock(cl))
	)

	require.NotNil(c)
	cl.OnNewTimer(time.Minute, timer).Once()
	timer.OnC(trigger).Once()
	timer.OnStop(true).Once().Run(func(mock.Arguments) {
		close(stopped)
	})

	for i := 0; i < 10; i++ {
		c.Submit(f)
	}

	trigger <- time.Time{}

	select {
	case <-stopped:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("The capacitor did not discharge properly")
	}

	cl.AssertExpectations(t)
	timer.AssertExpectations(t)
	assert.Equal(int32(1), atomic.LoadInt32(&calls))
}

func testCapacitorDischarge(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		stopped = make(chan struct{})
		calls   int32
		f       = func() {
			atomic.AddInt32(&calls, 1)
		}

		cl      = new(clocktest.Mock)
		timer   = new(clocktest.MockTimer)
		trigger = make(chan time.Time)
		c       = New(WithDelay(time.Minute), WithClock(cl))
	)

	require.NotNil(c)
	cl.OnNewTimer(time.Minute, timer).Once()
	timer.OnC(trigger).Once()
	timer.OnStop(true).Once().Run(func(mock.Arguments) {
		close(stopped)
	})

	for i := 0; i < 10; i++ {
		c.Submit(f)
	}

	c.Discharge()
	c.Discharge() // idempotent

	select {
	case <-stopped:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("The capacitor did not discharge properly")
	}

	cl.AssertExpectations(t)
	timer.AssertExpectations(t)
	assert.Equal(int32(1), atomic.LoadInt32(&calls))
}

func testCapacitorCancel(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		stopped = make(chan struct{})
		calls   int32
		f       = func() {
			atomic.AddInt32(&calls, 1)
		}

		cl      = new(clocktest.Mock)
		timer   = new(clocktest.MockTimer)
		trigger = make(chan time.Time)
		c       = New(WithDelay(time.Minute), WithClock(cl))
	)

	require.NotNil(c)
	cl.OnNewTimer(time.Minute, timer).Once()
	timer.OnC(trigger).Once()
	timer.OnStop(true).Once().Run(func(mock.Arguments) {
		close(stopped)
	})

	for i := 0; i < 10; i++ {
		c.Submit(f)
	}

	c.Cancel()
	c.Cancel() // idempotent

	select {
	case <-stopped:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("The capacitor did not discharge properly")
	}

	cl.AssertExpectations(t)
	timer.AssertExpectations(t)
	assert.Zero(atomic.LoadInt32(&calls))
}

func TestCapacitor(t *testing.T) {
	t.Run("Submit", testCapacitorSubmit)
	t.Run("Discharge", testCapacitorDischarge)
	t.Run("Cancel", testCapacitorCancel)
}
