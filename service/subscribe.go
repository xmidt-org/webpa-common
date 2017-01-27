package service

import (
	"github.com/Comcast/webpa-common/logging"
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
// The returned function can be called to cancel the subscription.  This returned cancellation
// function is idempotent.
func Subscribe(logger logging.Logger, watch Watch, subscription func([]string)) func() {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	logger.Debug("Creating subscription for %#v", watch)
	cancel := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Subscription ending due to panic: %s", r)
			}
		}()

		for {
			select {
			case <-cancel:
				logger.Info("Subscription cancel event received")
				return
			case <-watch.Event():
				logger.Debug("Watch event received")
				if watch.IsClosed() {
					logger.Info("Watch closed.  Subscription ending.")
					return
				}

				endpoints := watch.Endpoints()
				logger.Info("Updated endpoints: %v", endpoints)
				subscription(endpoints)
			}
		}
	}()

	return func() {
		defer func() {
			recover()
		}()

		logger.Debug("Subscription cancellation function called")
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
	accessor, _ := a.factory.New(endpoints)
	a.value.Store(accessor)
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

	// use update to initialze the atomic value
	subscription.update(watch.Endpoints())
	subscription.cancelFunc = Subscribe(o.logger(), watch, subscription.update)
	return subscription
}
