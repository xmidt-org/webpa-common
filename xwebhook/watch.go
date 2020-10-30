package xwebhook

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/xmidt-org/argus/chrysom"
)

// Watch is the interface for listening for webhook subcription updates.
// Updates represent the latest known list of subscriptions.
type Watch interface {
	Update([]Webhook)
}

// WatchFunc allows bare functions to pass as Watches.
type WatchFunc func([]Webhook)

func (f WatchFunc) Update(update []Webhook) {
	f(update)
}

// Config provides the different options for the initializing the wehbook service.
type Config struct {
	// Argus contains all the argus specific configurations
	Argus chrysom.ClientConfig

	// WatchUpdateInterval is the duration between each update to all watchers.
	WatchUpdateInterval time.Duration
}

func webhookListSizeWatch(s metrics.Gauge) Watch {
	return WatchFunc(func(webhooks []Webhook) {
		s.Set(float64(len(webhooks)))
	})
}
