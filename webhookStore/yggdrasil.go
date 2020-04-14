package webhookStore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"io/ioutil"
	"net/http"
	"time"
)

type YggdrasilConfig struct {
	Client       *http.Client
	Prefix       string
	PullInterval time.Duration
	Address      string
	Auth         Auth
}
type Auth struct {
	JWT   acquire.RemoteBearerTokenAcquirerOptions
	Basic string
}

type YggdrasilClient struct {
	client  *http.Client
	options *storeConfig
	config  YggdrasilConfig
	ticker  *time.Ticker
	auth    acquire.Acquirer
}

func CreateYggdrasilStore(config YggdrasilConfig, options ...Option) (*YggdrasilClient, error) {
	err := validateYggdrasilConfig(&config)
	if err != nil {
		return nil, err
	}
	auth, err := determineTokenAcquirer(config)
	if err != nil {
		return nil, err
	}
	clientStore := &YggdrasilClient{
		client: config.Client,
		options: &storeConfig{
			logger: logging.DefaultLogger(),
		},
		config: config,
		ticker: time.NewTicker(config.PullInterval),
		auth:   auth,
	}
	for _, o := range options {
		o(clientStore.options)
	}
	go func() {
		for range clientStore.ticker.C {
			if clientStore.options.listener != nil {
				hooks, err := clientStore.GetWebhook()
				if err == nil {
					clientStore.options.listener.Update(hooks)
				} else {
					logging.Error(clientStore.options.logger).Log(logging.MessageKey(), "failed to get webhooks ", logging.ErrorKey(), err)
				}
			}
		}
	}()
	return clientStore, nil
}

func validateYggdrasilConfig(config *YggdrasilConfig) error {
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	if config.Address == "" {
		return errors.New("yggdrasil address can't be empty")
	}
	if config.PullInterval == 0 {
		config.PullInterval = time.Second
	}
	if config.Prefix == "" {
		config.Prefix = "testing"
	}
	return nil
}
func determineTokenAcquirer(config YggdrasilConfig) (acquire.Acquirer, error) {
	defaultAcquirer := &acquire.DefaultAcquirer{}
	if config.Auth.JWT.AuthURL != "" && config.Auth.JWT.Buffer != 0 && config.Auth.JWT.Timeout != 0 {
		return acquire.NewRemoteBearerTokenAcquirer(config.Auth.JWT)
	}

	if config.Auth.Basic != "" {
		return acquire.NewFixedAuthAcquirer(config.Auth.Basic)
	}

	return defaultAcquirer, nil
}

func (c *YggdrasilClient) GetWebhook() ([]webhook.W, error) {
	hooks := []webhook.W{}
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/store/%s", c.config.Address, c.config.Prefix), nil)
	if err != nil {
		return []webhook.W{}, err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return []webhook.W{}, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return []webhook.W{}, err
	}
	if response.StatusCode == 404 {
		return []webhook.W{}, nil
	}
	if response.StatusCode != 200 {
		return []webhook.W{}, errors.New("failed to get webhooks, non 200 statuscode")
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []webhook.W{}, err
	}
	response.Body.Close()

	body := map[string]map[string]interface{}{}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return []webhook.W{}, err
	}

	for _, value := range body {
		data, err := json.Marshal(&value)
		if err != nil {
			continue
		}
		var hook webhook.W
		err = json.Unmarshal(data, &hook)
		if err != nil {
			continue
		}
		hooks = append(hooks, hook)
	}

	return hooks, nil
}

func (c *YggdrasilClient) Push(w webhook.W) error {
	id := base64.RawURLEncoding.EncodeToString([]byte(w.ID()))
	data, err := json.Marshal(&w)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/store/%s/%s", c.config.Address, c.config.Prefix, id), bytes.NewReader(data))
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("failed to push webhook, non 200 statuscode")
	}
	return nil
}

func (c *YggdrasilClient) Remove(id string) error {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/store/%s/%s", c.config.Address, c.config.Prefix, id), nil)
	if err != nil {
		return err
	}
	err = acquire.AddAuth(request, c.auth)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New("failed to delete webhook, non 200 statuscode")
	}
	return nil
}

func (c *YggdrasilClient) Stop(context context.Context) {
	c.ticker.Stop()
}

func (c *YggdrasilClient) SetListener(listener Listener) error {
	c.options.listener = listener
	return nil
}
