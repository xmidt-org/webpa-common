package xwebhook

import (
	"encoding/json"
	"time"

	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
)

// Service describes the core operations around webhook subscriptions.
// Initialize() provides a service ready to use and the controls around watching for updates.
type Service interface {
	// Add adds the given owned webhook to the current list of webhooks. If the operation
	// succeeds, it returns a non-empty ID for the webhook.
	Add(owner string, w *Webhook) (string, error)

	// AllWebhooks lists all the current webhooks for the given owner.
	// If an owner is not provided, all webhooks are returned.
	AllWebhooks(owner string) ([]Webhook, error)

	//TODO: we can technically support deletion as well if we wanted to.
}

type service struct {
	argus *chrysom.Client
}

func (s *service) Add(owner string, w *Webhook) (string, error) {
	item, err := webhookToItem(w)
	if err != nil {
		return "", err
	}
	return s.argus.Push(*item, owner)
}

func (s *service) AllWebhooks(owner string) ([]Webhook, error) {
	items, err := s.argus.GetItems(owner)
	if err != nil {
		return nil, err
	}
	webhooks := []Webhook{}
	for _, item := range items {
		webhook, err := itemToWebhook(&item)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, *webhook)
	}

	return webhooks, nil
}

func webhookToItem(w *Webhook) (*model.Item, error) {
	encodedWebhook, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(encodedWebhook, &data)
	if err != nil {
		return nil, err
	}

	return &model.Item{
		Identifier: w.Address,
		Data:       data,
	}, nil
}

func itemToWebhook(i *model.Item) (*Webhook, error) {
	w := new(Webhook)
	encodedWebhook, err := json.Marshal(i.Data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(encodedWebhook, w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func newService(cfg *Config) (Service, error) {
	argus, err := chrysom.CreateClient(cfg.Argus)
	if err != nil {
		return nil, err
	}
	svc := &service{
		argus: argus,
	}
	return svc, nil
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func Initialize(cfg *Config, p provider.Provider, watchers ...Watch) (Service, func(), error) {
	validateConfig(cfg)

	svc, err := newService(cfg)
	if err != nil {
		return nil, nil, err
	}

	watchers = append(watchers, webhookListSizeWatch(p.NewGauge(WebhookListSizeGauge)))

	stopWatchers := startWatchers(cfg.WatchUpdateInterval, p.NewCounter(PollCounter), svc, watchers...)
	return svc, stopWatchers, nil
}

func validateConfig(cfg *Config) {
	if cfg.WatchUpdateInterval == 0 {
		cfg.WatchUpdateInterval = time.Second * 5
	}
}
