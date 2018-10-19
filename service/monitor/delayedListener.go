package monitor

import "github.com/Comcast/webpa-common/capacitor"

// DelayedListener is a decorator for Listener that uses a capacitor to implement a grace period
// between service discovery events.
type DelayedListener struct {
	Listener  Listener
	Capacitor capacitor.Interface
}

func (dl DelayedListener) MonitorEvent(e Event) {
	dl.Capacitor.Submit(func() {
		dl.Listener.MonitorEvent(e)
	})
}
