package xwebhook

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

func setupWatchers(watchers Watch...) func() {
	//TODO: 
	return func() {}
}