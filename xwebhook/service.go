package xwebhook

import (
	"time"

	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/themis/config"
	"github.com/fatih/structs"

)

// Service describes the core operations around webhook subscriptions.
// Initialize() provides a service ready to use and the controls around watching for updates.
type Service interface {
	// Add adds the given owned webhook to the current list of webhooks.
	Add(owner string, w Webhook) error

	// AllWebhooks lists all the current webhooks for the given owner. 
	// If an owner is not provided, all webhooks are returned.
	AllWebhooks(owner string) ([]Webhook, error)
}

type service struct {
	argus *chrysom.Client
}

func(s *service) Add(owner string, w Webhook) error {
	item :=  webhookToItem(w)
	_, err := s.argus.Push(*item, owner)
	return err
}

func (s *service) AllWebhooks(owner string) ([]Webhook, error) {
	items, err := s.argus.GetItems(owner)
	if err != nil {
		return
	}
	var webhooks []Webhook
	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks
}

func webhookToItem(w Webhook) model.Item {
	return model.Item{
		Identifier: w.Address,
		Data: structs.Map(w),
	}
}

func itemToWebhook(i model.Item) (Webhook, error) {
	w := new(Webhook)
	tempBytes, err := json.Marshal(&i.Data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(tempBytes, w)
	if err != nil {
		return nil, err
	}
	return *w, nil
}

func newService(cfg Config) (Service, error){
	argus, err := chrysom.CreateClient(cfg.Argus)
	if err != nil {
		return nil, err
	}
	svc := service {
		argus: argus,
	}
	return svc, error
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates. 
func Initialize(key string, config config.KeyUnmarshaller, watchers Watch...) (Service, func(), error) {
	var cfg Config
	err := config.UnmarshalKey(key, &cfg)
	if err != nil {
		return nil, nil, err
	}

	svc  := newService(cfg Config)
	stopWatchers := startWatchers(watchers)
	return svc, stopWatchers, nil
}
