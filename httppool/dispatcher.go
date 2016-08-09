package httppool

import (
	"io"
	"net/http"
)

// transactionHandler defines the methods required of something that actually
// handles HTTP transactions.  http.Client satisfies this interface.
type transactionHandler interface {
	// Do synchronously handles the HTTP transaction.  Any type that supplies
	// this method may be used with this infrastructure.
	Do(*http.Request) (*http.Response, error)
}

// Task is a constructor function type that creates http.Request objects
// A task is used, rather than a request directly, to allow lazy instantiation
// of requests at the time the request is to be sent.
type Task func() (*http.Request, error)

// RequestTask allows an already-formed http.Request to be used as a Task.
func RequestTask(request *http.Request) Task {
	return Task(func() (*http.Request, error) {
		return request, nil
	})
}

// Dispatcher represents anything that can receive and process tasks.
type Dispatcher interface {
	// Send uses the task to create a request and sends that along to a server.
	// This method may be asynchronous or synchronous, depending on the underlying implementation.
	Send(Task) error
}

// DispatchCloser is a Dispatcher that can be closed.
type DispatchCloser interface {
	Dispatcher
	io.Closer
}
