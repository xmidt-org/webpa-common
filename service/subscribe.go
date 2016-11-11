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

// Subscription represents something that is receiving events from a watch and updating
// its state.
type Subscription interface {
	// Cancel removes this subscription from the underlying infrastructure.  No further
	// updates will occur, but this subscription's state will still be usable.
	// This method is idempotent.
	Cancel()
}

// AccessorSubscription represents an Accessor whose state changes as the result
// of events via a subscription.
type AccessorSubscription interface {
	Accessor
	Subscription
}

// accessorSubscription is the internal implementation of AccessorSubscription
type accessorSubscription struct {
	factory AccessorFactory
	value   atomic.Value
	cancel  chan struct{}
}

func (a *accessorSubscription) Cancel() {
	defer func() {
		recover()
	}()

	close(a.cancel)
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
func NewAccessorSubscription(o *Options, watch Watch, factory AccessorFactory) AccessorSubscription {
	logger := o.logger()
	if factory == nil {
		factory = NewAccessorFactory(o)
	}

	subscription := &accessorSubscription{
		factory: factory,
		cancel:  make(chan struct{}),
	}

	// load the initial accessor
	initialEndpoints := watch.Endpoints()
	logger.Info("Initial discovered endpoints: %s", initialEndpoints)
	subscription.update(initialEndpoints)

	// spawn a goroutine that updates the subscription in response
	// to watch events.
	go func() {
		for {
			select {
			case <-subscription.cancel:
				logger.Info("Subscription cancelled")
				return
			case <-watch.Event():
				if watch.IsClosed() {
					logger.Info("Subscription ending due to watch being closed")
					return
				}

				updatedEndpoints := watch.Endpoints()
				logger.Info("Updated endpoints: %s", updatedEndpoints)
				subscription.update(updatedEndpoints)
			}
		}
	}()

	return subscription
}
