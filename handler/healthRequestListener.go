package handler

import (
	"github.com/Comcast/webpa-common/health"
	"net/http"
)

const (
	TotalRequestsReceived            health.Stat = "TotalRequestsReceived"
	TotalRequestSuccessfullyServiced health.Stat = "TotalRequestSuccessfullyServiced"
	TotalRequestDenied               health.Stat = "TotalRequestDenied"
)

// healthEventSink is the internal, expected interface that health request events are dispatched to
type healthEventSink interface {
	SendEvent(health.HealthFunc)
}

// NewHealthRequestListener returns a new RequestListener which dispatches request stats
func NewHealthRequestListener(sink healthEventSink) RequestListener {
	return &healthRequestListener{sink}
}

// healthRequestListener is a handler.RequestListener that updates the health-related stats.
type healthRequestListener struct {
	sink healthEventSink
}

func (listener *healthRequestListener) RequestReceived(request *http.Request) {
	listener.sink.SendEvent(
		health.Inc(TotalRequestsReceived, 1),
	)
}

func (listener *healthRequestListener) RequestCompleted(statusCode int, request *http.Request) {
	if statusCode < 400 {
		listener.sink.SendEvent(
			health.Inc(TotalRequestSuccessfullyServiced, 1),
		)
	} else {
		listener.sink.SendEvent(
			health.Inc(TotalRequestDenied, 1),
		)
	}
}
