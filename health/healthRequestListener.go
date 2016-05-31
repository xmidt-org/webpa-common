package health

import (
	"net/http"
)

// HealthRequestListener is a handler.RequestListener that updates the health-related stats.
type HealthRequestListener struct {
	monitor *Health
}

func (listener *HealthRequestListener) RequestReceived(request *http.Request) {
	listener.monitor.SendEvent(
		Inc(TotalRequestsReceived, 1),
	)
}

func (listener *HealthRequestListener) RequestCompleted(statusCode int, request *http.Request) {
	if statusCode < 400 {
		listener.monitor.SendEvent(
			Inc(TotalRequestSuccessfullyServiced, 1),
		)
	} else {
		listener.monitor.SendEvent(
			Inc(TotalRequestDenied, 1),
		)
	}
}
