package capacitor

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/clock/clocktest"
	"github.com/stretchr/testify/assert"
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
