package capacitor

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/clock"
)

const DefaultDelay time.Duration = time.Second

type Interface interface {
	Submit(func())
	Discharge()
	Cancel()
}

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
