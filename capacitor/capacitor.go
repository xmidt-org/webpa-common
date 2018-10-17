package capacitor

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/clock"
)

type Interface interface {
	Submit(func())
	Discharge()
	Cancel()
}

func New() Interface {
	c := &capacitor{
		delay: time.Second,
		c:     clock.System(),
	}

	return c
}

type delayer struct {
	current   atomic.Value
	timer     clock.Timer
	terminate chan bool
	reset     func()
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

func (d *delayer) run() {
	defer d.timer.Stop()
	defer d.reset()

	for {
		select {
		case <-d.timer.C():
			select {
			case discharge := <-d.terminate:
				if discharge {
					d.execute()
				}

			default:
				d.execute()
			}

			return

		case discharge := <-d.terminate:
			if discharge {
				d.execute()
			}
		}
	}
}

type capacitor struct {
	lock  sync.Mutex
	delay time.Duration
	c     clock.Interface
	d     *delayer
}

// reset produces a closure that a given delayer must call to clean up the enclosing capacitor.
// The returned closure atomically sets the delayer to nil if and only if it matched the given delayer.
// This allows for barging between the public interface methods.
func (c *capacitor) reset(d *delayer) func() {
	return func() {
		c.lock.Lock()
		if c.d == d {
			c.d = nil
		}
		c.lock.Unlock()
	}
}

func (c *capacitor) Submit(v func()) {
	c.lock.Lock()
	if c.d == nil {
		c.d = &delayer{
			terminate: make(chan bool, 1),
			timer:     c.c.NewTimer(c.delay),
		}

		c.d.current.Store(v)
		c.d.reset = c.reset(c.d)
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
