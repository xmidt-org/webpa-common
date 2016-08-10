package httppool

import (
	"io"
	"net/http"
)

// Consumer is a function type which is invoked with the results of an HTTP transaction.
// Normally, this function will be invoked asynchronously.
//
// Dispatchers will never invoke this function if either parameter is nil.
type Consumer func(*http.Response, *http.Request)

// Task is a constructor function type that creates http.Request objects
// A task is used, rather than a request directly, to allow lazy instantiation
// of requests at the time the request is to be sent.
//
// Each Task may optionally return a Consumer.  If non-nil, this function is
// invoked with the request/response pair.  The Dispatcher will always cleanup
// the http.Response, regardless of whether the Consumer does anything with
// the response body.
type Task func() (*http.Request, Consumer, error)

// RequestTask allows an already-formed http.Request to be used as a Task.
func RequestTask(request *http.Request, consumer Consumer) Task {
	return Task(func() (*http.Request, Consumer, error) {
		return request, consumer, nil
	})
}

// Dispatcher represents anything that can receive and process tasks.
type Dispatcher interface {
	// Send uses the task to create a request and sends that along to a server.
	// This method may be asynchronous or synchronous, depending on the underlying implementation.
	// A caller of this method will block until the Dispatcher is able to handle the task.
	Send(Task) error

	// Offer is similar to Send, except that the Dispatcher can reject the task.  A false
	// return value indicates that the task was not executed, regardless of the error return value.
	// Typically, Offer will reject tasks because an underlying queue or buffer is full.
	Offer(Task) (bool, error)
}

// DispatchCloser is a Dispatcher that can be closed.  The Close() method implemented
// by any DispatchCloser must be idempotent.  It should not panic when called multiple times.
type DispatchCloser interface {
	Dispatcher
	io.Closer
}
