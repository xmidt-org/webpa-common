package xwebhook

import (
	"time"

	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/themis/config"
)

// Service describes the core operations around webhook subscriptions.
// Initialize() provides a service ready to use and the controls around watching for updates.
type Service interface {
	// Subscribe adds the given subscription to the internal shared store.
	Subscribe(*Subscription)

	// AllSubscriptions lists all the current webhook subscriptions.
	AllSubscriptions(owner string) []Subscription
}

// Watch is the interface for listening for webhook subcription updates.
// Updates represent the latest known list of subscriptions.
type Watch interface {
	Update([]Subscription)
}

// WatchFunc allows bare functions to pass as Watches.
type WatchFunc func([]Subscription)

func (f WatchFunc) Update(update []Subscription) {
	f(update)
}

// Config provides the different options for the initializing the wehbook service.
type Config struct {
	Argus          chrysom.ClientConfig
	UpdateInterval time.Duration
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates. 
func Initialize(key string, config config.KeyUnmarshaller, p provider.Provider, watchers Watch...) (svc Service, stopUpdating func(), err error) {
	//TODO: do the work
	return nil, nil, nil
}
