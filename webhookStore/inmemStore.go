package webhookStore

import (
	"context"
	"errors"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"sync"
	"time"
)

var (
	errNoHookAvailable = errors.New("no webhook for key")
)

type envelope struct {
	creation time.Time
	hook     webhook.W
}

type InMem struct {
	hooks   map[string]envelope
	lock    sync.RWMutex
	config  InMemConfig
	options *storeConfig
	stop    chan struct{}
}

func (inMem *InMem) Remove(id string) error {
	// update the store if there is no backend.
	// if it is set. On List() will update the inmem data set
	if inMem.options.backend == nil {
		inMem.lock.Lock()
		delete(inMem.hooks, id)
		inMem.lock.Unlock()
		// update listener
		if inMem.options.listener != nil {
			hooks, _ := inMem.GetWebhook()
			inMem.options.listener.Update(hooks)
		}
		return nil
	}
	return inMem.options.backend.Remove(id)
}

func (inMem *InMem) Stop(ctx context.Context) {
	close(inMem.stop)
	if inMem.options.backend != nil {
		inMem.options.backend.Stop(ctx)
	}
}

func (inMem *InMem) GetWebhook() ([]webhook.W, error) {
	if inMem.options.backend != nil {
		if reader, ok := inMem.options.backend.(Reader); ok {
			return reader.GetWebhook()
		}
	}
	inMem.lock.RLock()
	data := []webhook.W{}
	for _, value := range inMem.hooks {
		if time.Now().Before(value.creation.Add(inMem.config.TTL)) {
			data = append(data, value.hook)
		}
	}
	inMem.lock.RUnlock()
	return data, nil
}

func (inMem *InMem) Update(hooks []webhook.W) {
	// update inmem
	if inMem.options.listener != nil {
		inMem.hooks = map[string]envelope{}
		for _, elem := range hooks {
			inMem.hooks[elem.ID()] = envelope{
				creation: time.Now(),
				hook:     elem,
			}
		}
	}
	// notify listener
	if inMem.options.listener != nil {
		inMem.options.listener.Update(hooks)
	}
}

func (inMem *InMem) Push(w webhook.W) error {
	// update the store if there is no backend.
	// if it is set. On List() will update the inmem data set
	updateStructure := func() {
		inMem.lock.Lock()
		inMem.hooks[w.ID()] = envelope{
			creation: time.Now(),
			hook:     w,
		}
		inMem.lock.Unlock()
		// update listener
		if inMem.options.listener != nil {
			hooks, _ := inMem.GetWebhook()
			inMem.options.listener.Update(hooks)
		}
	}

	if inMem.options.backend == nil {
		updateStructure()
		return nil
	}
	// if backend is not a listener or inMem is not a listener of the backend.
	// update the internal structure
	if listener, ok := inMem.options.backend.(Listener); !ok || listener != inMem {
		updateStructure()
	}
	return inMem.options.backend.Push(w)
}

// CleanUp will free remove old webhooks.
func (inMem *InMem) CleanUp() {
	inMem.lock.Lock()
	for key, value := range inMem.hooks {
		if value.creation.Add(inMem.config.TTL).After(time.Now()) {
			go inMem.Remove(key)
		}
	}
	inMem.lock.Unlock()
}

type InMemConfig struct {
	TTL           time.Duration
	CheckInterval time.Duration
}

const (
	defaultTTL           = time.Minute * 5
	defaultCheckInterval = time.Minute
)

func validateConfig(config InMemConfig) InMemConfig {
	if config.TTL.Nanoseconds() == 0 {
		config.TTL = defaultTTL
	}
	if config.CheckInterval.Nanoseconds() == int64(0) {
		config.CheckInterval = defaultCheckInterval
	}
	return config
}

// CreateInMemStore will create an inmemory storage that will handle ttl of webhooks.
// listner and back and optional and can be nil
func CreateInMemStore(config InMemConfig, options ...Option) *InMem {
	inMem := &InMem{
		hooks:  map[string]envelope{},
		config: validateConfig(config),
		stop:   make(chan struct{}),
	}
	inMem.options = &storeConfig{
		logger: logging.DefaultLogger(),
		self:   inMem,
	}

	for _, o := range options {
		o(inMem.options)
	}

	ticker := time.NewTicker(inMem.config.CheckInterval)
	go func() {
		for {
			select {
			case <-inMem.stop:
				return
			case <-ticker.C:
				inMem.CleanUp()
			}
		}
	}()
	return inMem
}
