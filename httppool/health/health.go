package health

import (
	"github.com/xmidt-org/webpa-common/health"
	"github.com/xmidt-org/webpa-common/httppool"
)

const (
	TotalNotificationsQueued    health.Stat = "TotalNotificationsQueued"
	TotalNotificationsRejected  health.Stat = "TotalNotificationsRejected"
	TotalNotificationsSucceeded health.Stat = "TotalNotificationsSucceeded"
	TotalNotificationsFailed    health.Stat = "TotalNotificationsFailed"
)

// listener is an internal httppool.Listener that delegates to the given
// health monitor
type listener struct {
	monitor health.Monitor
}

func (l *listener) On(event httppool.Event) {
	switch event.Type() {
	case httppool.EventTypeQueue:
		l.monitor.SendEvent(health.Inc(TotalNotificationsQueued, 1))
	case httppool.EventTypeReject:
		l.monitor.SendEvent(health.Inc(TotalNotificationsRejected, 1))
	case httppool.EventTypeFinish:
		if event.Err() != nil {
			l.monitor.SendEvent(health.Inc(TotalNotificationsFailed, 1))
		} else {
			l.monitor.SendEvent(health.Inc(TotalNotificationsSucceeded, 1))
		}
	}
}

// Listener constructs an httppool.Listener that dispatches to a health Monitor
func Listener(monitor health.Monitor) httppool.Listener {
	return &listener{monitor}
}
