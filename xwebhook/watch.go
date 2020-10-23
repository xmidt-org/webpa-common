package xwebhook

import "time"

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
	Argus          chrysom.ClientConfig

	// WatchUpdateInterval is the duration between each update to all watchers.
	WatchUpdateInterval time.Duration
}

func startWatchers(updateInterval time.Duration, svc Service, watchers Watch...) func() {
	ticker := time.NewTicker(updateInterval)

	go func() {
		for range ticker.C {
			webhooks, err := svc.AllWebhooks("") 
			if err != nil {
				continue
			}

			for _, watcher := range watchers {
				watcher.Update(webhooks)
			}
		}
	}()

	return func() {
		ticker.Stop()
	}
}