package service

import (
	"sync/atomic"
	"testing"
)

// TestWatch is a Watch implementation designed to test go.serversets event loops.
// Event, IsClosed, and Endpoints are expected to be called within an event processing
// goroutine.  NextEndpoints is expected to be called by test code.  Close may be
// called by any goroutine at any time, and is idempotent.
type TestWatch struct {
	t         *testing.T
	events    chan chan struct{}
	endpoints chan []string
	closed    uint32
}

func (tw *TestWatch) Close() {
	if atomic.CompareAndSwapUint32(&tw.closed, 0, 1) {
		// extract the latest event, and close it and the events channel
		// if the enclosing test is testing panic or error behavior,
		// there may not be a last event
		select {
		case event := <-tw.events:
			close(event)
		default:
		}
	}
}

func (tw *TestWatch) IsClosed() bool {
	return atomic.LoadUint32(&tw.closed) != 0
}

// Event enqueues the next event channel to return.  This event channel
// is dequeued by NextEndpoints.
func (tw *TestWatch) Event() <-chan struct{} {
	if tw.IsClosed() {
		tw.t.Error("The watch has been closed")
		return nil
	}

	event := make(chan struct{}, 1)
	tw.events <- event
	return event
}

// Endpoints returns the current endpoints, blocking until one slice of endpoints
// is available
func (tw *TestWatch) Endpoints() []string {
	if tw.IsClosed() {
		tw.t.Error("The watch has been closed")
		return nil
	}

	// block waiting for the next slice of endpoints
	return <-tw.endpoints
}

// NextEndpoints is used by test code to enqueue the next set of endpoints.
// This method uses OnEvent to enqueue the given endpoints.  This method should
// not be used when testing behavior that will not result in a call to Endpoints,
// such as panic or error behavior.
func (tw *TestWatch) NextEndpoints(endpoints []string) {
	tw.OnEvent(func() {
		tw.endpoints <- endpoints
	})
}

// OnEvent waits until another goroutine calls Event.  That event channel is then triggered,
// and the given operation is executed.
func (tw *TestWatch) OnEvent(operation func()) {
	if tw.IsClosed() {
		tw.t.Fatal("The watch has been closed")
		return
	}

	// wait until some goroutine calls Event
	if event := <-tw.events; event != nil {
		// closing the event signals a real Watch sending an event
		close(event)

		// invoke the operation after the event is triggered
		operation()
	}
}

func NewTestWatch(t *testing.T) *TestWatch {
	return &TestWatch{
		t:         t,
		events:    make(chan chan struct{}, 1),
		endpoints: make(chan []string),
	}
}
