package devicehealth

import (
	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/health"
)

const (
	DeviceCount                      health.Stat = "DeviceCount"
	TotalWRPRequestResponseProcessed health.Stat = "TotalWRPRequestResponseProcessed"
	TotalPongMessagesReceived        health.Stat = "TotalPongMessagesReceived"
	TotalConnectionEvents            health.Stat = "TotalConnectionEvents"
	TotalDisconnectionEvents         health.Stat = "TotalDisconnectionEvents"
)

// Listener provides a Device.Listener that dispatches health statistics
type Listener struct {
	Dispatcher health.Dispatcher
}

func (l *Listener) OnDeviceEvent(e *device.Event) {
	switch e.Type {
	case device.Connect:
		l.Dispatcher.SendEvent(func(s health.Stats) {
			s[DeviceCount] += 1
			s[TotalConnectionEvents] += 1
		})

	case device.Disconnect:
		l.Dispatcher.SendEvent(func(s health.Stats) {
			s[DeviceCount] -= 1
			s[TotalDisconnectionEvents] += 1
		})

	case device.TransactionComplete:
		l.Dispatcher.SendEvent(func(s health.Stats) {
			s[TotalWRPRequestResponseProcessed] += 1
		})

	case device.Pong:
		l.Dispatcher.SendEvent(func(s health.Stats) {
			s[TotalPongMessagesReceived] += 1
		})
	}
}
