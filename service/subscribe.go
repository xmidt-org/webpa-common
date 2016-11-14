package service

import (
	"github.com/strava/go.serversets"
	"sync/atomic"
)

// Watch is the subset of methods required by this package that *serversets.Watch implements
type Watch interface {
	IsClosed() bool
	Event() <-chan struct{}
	Endpoints() []string
}

var _ Watch = (*serversets.Watch)(nil)

// Subscribe consumes watch events and invokes a subscription function with the endpoints.
// An initial subscription invocation with the initial set of endpoints is made, then
// a goroutine is spawn to watch for changes and dispatch updates to the subscription.
//
// The returned function can be called to cancel the subscription.  This returned cancellation
// function is idempotent.
func Subscribe(watch Watch, subscription func([]string)) func() {
	cancel := make(chan struct{})
	go func() {
		// send the initial endpoints first
		subscription(watch.Endpoints())

		for {
			select {
			case <-cancel:
				return
			case <-watch.Event():
				if watch.IsClosed() {
					return
				}

				subscription(watch.Endpoints())
			}
		}
	}()

	return func() {
		defer func() {
			recover()
		}()

		close(cancel)
	}
}

// AccessorSubscription represents an Accessor whose state changes as the result
// of events via a subscription.
type AccessorSubscription interface {
	Accessor

	// Cancel removes this subscription from the underlying infrastructure.  No further
	// updates will occur, but this subscription's state will still be usable.
	// This method is idempotent.
	Cancel()
}

// accessorSubscription is the internal implementation of AccessorSubscription
type accessorSubscription struct {
	factory    AccessorFactory
	value      atomic.Value
	cancelFunc func()
}

func (a *accessorSubscription) Cancel() {
	a.cancelFunc()
}

func (a *accessorSubscription) Get(key []byte) (string, error) {
	return a.value.Load().(Accessor).Get(key)
}

func (a *accessorSubscription) update(endpoints []string) {
	a.value.Store(a.factory.New(endpoints))
}

// NewAccessorSubscription subscribes to a watch and updates an atomic Accessor in response
// to updated service endpoints.  The returned object is fully initialized and can be used
// to access endpoints immediately.  In addition, the subscription can be cancelled at any time.
// If the underlying service discovery infrastructure is shutdown, the subscription will no
// longer receive updates but can continue to be used in its stale state.
func NewAccessorSubscription(watch Watch, factory AccessorFactory, o *Options) AccessorSubscription {
	if factory == nil {
		factory = NewAccessorFactory(o)
	}

	subscription := &accessorSubscription{
		factory: factory,
	}

	subscription.cancelFunc = Subscribe(watch, subscription.update)
	return subscription
}
