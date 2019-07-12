package capacitor

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xmidt-org/webpa-common/clock"
)

// DefaultDelay is the default time a capacitor waits to execute the most recently
// submitted function
const DefaultDelay time.Duration = time.Second

// Interface represents a capacitor of function calls which will discharge after
// a configurable period of time.
type Interface interface {
	// Submit submits a function for execution.  The function will not be executed immediately.
	// Instead, after a configurable period of time, the most recent function passed to Submit will
	// be executed.  The previous ones are ignored.
	Submit(func())

	// Discharge forcibly discharges this capacitor.  The most recent function passed to Submit is
	// executed, and the internal state is reset so that the next call to Submit will start the
	// process of delaying function calls all over again.
	Discharge()

	// Cancel terminates any waiting function call without executing it.  As with Discharge, the
	// internal state is reset so that Submit calls will delay functions as normal again.
	Cancel()
}

// Option represents a configuration option for a capacitor
type Option func(*capacitor)

func WithDelay(d time.Duration) Option {
	return func(c *capacitor) {
		if d > 0 {
			c.delay = d
		} else {
			c.delay = DefaultDelay
		}
	}
}

func WithClock(cl clock.Interface) Option {
	return func(c *capacitor) {
		if cl != nil {
			c.c = cl
		} else {
			c.c = clock.System()
		}
	}
}

// New creates a capacitor with the given options.
func New(o ...Option) Interface {
	c := &capacitor{
		delay: DefaultDelay,
		c:     clock.System(),
	}

	for _, f := range o {
		f(c)
	}

	return c
}

// delayer is the internal job type that holds the context for a single, delayed
// charge of the capacitor
type delayer struct {
	current   atomic.Value
	timer     <-chan time.Time
	terminate chan bool
	cleanup   func()
}

func (d *delayer) discharge() {
	d.terminate <- true
}

func (d *delayer) cancel() {
	d.terminate <- false
}

func (d *delayer) execute() {
	if f, ok := d.current.Load().(func()); f != nil && ok {
		f()
	}
}

// run is called as a goroutine and will exit when either the timer channel
// is signalled or the terminate channel receives a value.
func (d *delayer) run() {
	defer d.cleanup()

	select {
	case <-d.timer:
		// since the timer can fire at the same time as Discharge or Cancel,
		// we want to make sure that any type of explicit termination trumps the timer
		select {
		case discharge := <-d.terminate:
			if discharge {
				d.execute()
			}

		default:
			d.execute()
		}

	case discharge := <-d.terminate:
		if discharge {
			d.execute()
		}
	}
}

// capacitor implements Interface, and provides an atomically updated delayer job
type capacitor struct {
	lock  sync.Mutex
	delay time.Duration
	c     clock.Interface
	d     *delayer
}

func (c *capacitor) Submit(v func()) {
	c.lock.Lock()
	if c.d == nil {
		var (
			t = c.c.NewTimer(c.delay)
			d = &delayer{
				terminate: make(chan bool, 1),
				timer:     t.C(),
			}
		)

		d.current.Store(v)

		// create a cleanup closure that stops the timer and
		// ensures that the given delayer is cleared, allowing
		// for barging.
		d.cleanup = func() {
			t.Stop()
			c.lock.Lock()
			if c.d == d {
				c.d = nil
			}
			c.lock.Unlock()
		}

		c.d = d
		go c.d.run()
	} else {
		c.d.current.Store(v)
	}

	c.lock.Unlock()
}

func (c *capacitor) Discharge() {
	c.lock.Lock()
	if c.d != nil {
		c.d.discharge()
		c.d = nil
	}

	c.lock.Unlock()
}

func (c *capacitor) Cancel() {
	c.lock.Lock()
	if c.d != nil {
		c.d.cancel()
		c.d = nil
	}

	c.lock.Unlock()
}
