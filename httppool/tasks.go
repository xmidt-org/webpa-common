package httppool

import (
	"errors"
	"net/http"
	"time"
)

var (
	ErrorTaskExpired  = errors.New("Task has expired")
	ErrorTaskFiltered = errors.New("Task has been rejected by a RequestFilter")
)

// RequestTask allows an already-formed http.Request to be used as a Task.
func RequestTask(request *http.Request, consumer Consumer) Task {
	return Task(func() (*http.Request, Consumer, error) {
		return request, consumer, nil
	})
}

// PerishableTask is a constructor that returns a decorator Task that returns
// ErrorTaskExpired if the given expiry has been reached.
func PerishableTask(expiry time.Time, delegate Task) Task {
	return func() (*http.Request, Consumer, error) {
		if expiry.Before(time.Now()) {
			return nil, nil, ErrorTaskExpired
		}

		return delegate()
	}
}

// RequestFilter provides a way to accept or reject requests.  This
// is useful to determine if a task should proceed based on current
// conditions of the application, e.g. queues backed up, chatty clients, etc.
type RequestFilter interface {
	Accept(*http.Request) bool
}

// FilteredTask is a constructor that produces a decorator Task that
// checks if a given delegate's request should be allowed to proceed.
// If the delegate panics or returns an error, the returned Task will
// do the same.
func FilteredTask(filter RequestFilter, delegate Task) Task {
	return func() (*http.Request, Consumer, error) {
		request, consumer, err := delegate()
		if request == nil || err != nil {
			return nil, nil, err
		} else if !filter.Accept(request) {
			return nil, nil, ErrorTaskFiltered
		}

		return request, consumer, nil
	}
}
