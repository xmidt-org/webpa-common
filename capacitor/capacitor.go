package capacitor

import (
	"sync"
	"sync/atomic"
	"time"
)

type Interface interface {
	Submit(func())
	Discharge()
	Cancel()
}

func New() Interface {
	c := &capacitor{
		delay: time.Second,
	}

	return c
}

type delayer struct {
	current   atomic.Value
	terminate chan bool
}

func (d *delayer) cancel(discharge bool) {
	d.terminate <- discharge
}

func (d *delayer) execute() {
	if f, ok := d.current.Load().(func()); f != nil && ok {
		f()
	}
}

func (d *delayer) run(timer <-chan time.Time, stop func() bool) {
	defer stop()
	for {
		select {
		case <-timer:
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
	d     *delayer
}

func (c *capacitor) Submit(v func()) {
	c.lock.Lock()
	if c.d == nil {
		c.d = &delayer{
			terminate: make(chan bool, 1),
		}

		timer := time.NewTimer(c.delay)
		go c.d.run(timer.C, timer.Stop)
	}

	c.d.current.Store(v)
	c.lock.Unlock()
}

func (c *capacitor) Discharge() {
	c.lock.Lock()
	if c.d != nil {
		c.d.cancel(true)
		c.d = nil
	}

	c.lock.Unlock()
}

func (c *capacitor) Cancel() {
	c.lock.Lock()
	if c.d != nil {
		c.d.cancel(false)
		c.d = nil
	}

	c.lock.Unlock()
}
