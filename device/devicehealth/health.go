package devicehealth

import (
	"github.com/xmidt-org/webpa-common/v2/device"
	"github.com/xmidt-org/webpa-common/v2/health"
)

const (
	DeviceCount                      health.Stat = "DeviceCount"
	TotalWRPRequestResponseProcessed health.Stat = "TotalWRPRequestResponseProcessed"
	TotalPingMessagesReceived        health.Stat = "TotalPingMessagesReceived"
	TotalPongMessagesReceived        health.Stat = "TotalPongMessagesReceived"
	TotalConnectionEvents            health.Stat = "TotalConnectionEvents"
	TotalDisconnectionEvents         health.Stat = "TotalDisconnectionEvents"
)

// Options is an array of all the health Options exposed via this package
var Options = []health.Option{
	DeviceCount,
	TotalWRPRequestResponseProcessed,
	TotalPingMessagesReceived,
	TotalPongMessagesReceived,
	TotalConnectionEvents,
	TotalDisconnectionEvents,
}

// Listener provides a device.Listener that dispatches health statistics
type Listener struct {
	Dispatcher health.Dispatcher
}

// OnDeviceEvent is a device.Listener that will dispatched health events to the configured
// health Dispatcher.
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
	}
}
