package webhookStore

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/webhook"
)

type Pusher interface {
	// Push applies user configurable for registering a webhook
	// i.e. updated the storage with said webhook.
	Push(w webhook.W) error

	// Remove will remove the webhook from the store
	Remove(id string) error

	// Stop will stop all threads and cleanup any necessary resources
	Stop(context context.Context)
}

type Listener interface {
	// Update is called when we get changes to our webhook listeners with either
	// additions, or updates.
	//
	// The list of hooks must contain only the current webhooks.
	Update(hooks []webhook.W)
}

type ListenerFunc func(hooks []webhook.W)

func (listner ListenerFunc) Update(hooks []webhook.W) {
	listner(hooks)
}

type Reader interface {
	// GetWebhook will return all the current webhooks or an error.
	GetWebhook() ([]webhook.W, error)
}

type ConfigureListener interface {
	// SetListener will attempt to set the lister.
	SetListener(listener Listener) error
}

type storeConfig struct {
	logger   log.Logger
	backend  Pusher
	listener Listener
}

// Option is the function used to configure a store.
type Option func(r *storeConfig)

// WithLogger sets a logger to use for the store.
func WithLogger(logger log.Logger) Option {
	return func(r *storeConfig) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// WithStorage sets a Pusher to use for the store.
func WithStorage(pusher Pusher) Option {
	return func(r *storeConfig) {
		if pusher != nil {
			r.backend = pusher
		}
	}
}

// WithListener sets a Listener to use for the store.
func WithListener(listener Listener) Option {
	return func(r *storeConfig) {
		if listener != nil {
			r.listener = listener
		}
	}
}