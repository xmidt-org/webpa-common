package xwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	fmt.Printf("\n\nPushing item %v\n", item)
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
		Identifier: w.Config.URL,
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

func newService(argus *chrysom.Client) (Service, error) {
	svc := &service{
		argus: argus,
	}
	return svc, nil
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func Initialize(cfg *Config, watches ...Watch) (Service, func(), error) {
	validateConfig(cfg)

	watches = append(watches, webhookListSizeWatch(cfg.Argus.MetricsProvider.NewGauge(WebhookListSizeGauge)))

	cfg.Argus.Listener = createArgusListener(watches...)

	argus, err := chrysom.CreateClient(cfg.Argus)
	if err != nil {
		return nil, nil, err
	}

	svc, err := newService(argus)
	if err != nil {
		return nil, nil, err
	}

	argus.Start(context.Background())

	return svc, func() { argus.Stop(context.Background()) }, nil
}

func createArgusListener(watches ...Watch) chrysom.Listener {
	if len(watches) < 1 {
		return nil
	}
	return chrysom.ListenerFunc(func(items []model.Item) {
		webhooks := itemsToWebhooks(items)
		for _, watch := range watches {
			watch.Update(webhooks)
		}
	})
}

func itemsToWebhooks(items []model.Item) []Webhook {
	webhooks := []Webhook{}
	for _, item := range items {
		webhook, err := itemToWebhook(&item)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, *webhook)
	}
	return webhooks
}

func validateConfig(cfg *Config) {
	if cfg.WatchUpdateInterval == 0 {
		cfg.WatchUpdateInterval = time.Second * 5
	}
}
