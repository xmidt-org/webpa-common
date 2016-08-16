package httppool

type EventType int

const (
	// EventTypeQueue indicates that a task has been successfully queued
	EventTypeQueue EventType = iota

	// EventTypeReject indicates that a task was not queued, either because the
	// queue was full or because the pool was closed.
	EventTypeReject

	// EventTypeStart is sent when a task has been dequeued and is about to be processed
	EventTypeStart

	// EventTypeFinish indicates that a task has finished
	EventTypeFinish
)

// Event represents an interesting occurrence in an httppool
type Event interface {
	// Type is the type of this event
	Type() EventType

	// Error stores any error that occurred as part of this event
	Err() error
}

// event is the internal implementation of Event
type event struct {
	eventType  EventType
	eventError error
}

func (e *event) Type() EventType {
	return e.eventType
}

func (e *event) Err() error {
	return e.eventError
}

// Listener is a consumer of Events
type Listener interface {
	// On is a callback method for events
	On(Event)
}
