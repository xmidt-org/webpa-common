package webhookStore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	stdLog "log"
	"os"
)

type ConsulConfig struct {
	Client       *api.Client
	WriteOptions api.WriteOptions
	Prefix       string
}

type Client struct {
	client       *api.Client
	options      *storeConfig
	writeOptions api.WriteOptions
	readOptions  api.QueryOptions
	keyPrefix    string
	plan         *watch.Plan
}

// CreateInMemStore will create an inmemory storage that will handle ttl of webhooks.
// listner and back and optional and can be nil
func CreateConsulStore(config ConsulConfig, options ...Option) *Client {
	clientStore := &Client{
		client: config.Client,
		options: &storeConfig{
			logger: logging.DefaultLogger(),
		},
		keyPrefix: config.Prefix,
	}
	for _, o := range options {
		o(clientStore.options)
	}
	// onstart update listeners of current webhooks
	if clientStore.options.listener != nil {
		// TODO:// retry on err
		hooks, err := clientStore.GetWebhook()
		if err != nil {
			logging.Error(clientStore.options.logger).Log(logging.MessageKey(), "failed to unmarshal webhook", logging.ErrorKey(), err)
		} else {
			clientStore.options.listener.Update(hooks)
		}
	}
	// start watch for
	plan, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": clientStore.keyPrefix + "/"})

	if err != nil {
		logging.Error(clientStore.options.logger).Log(logging.MessageKey(), "failed create plan", logging.ErrorKey(), err)
		return nil
	}
	plan.Handler = clientStore.handlePlanCallback
	go func() {
		stdLogger := stdLog.New(os.Stdout, "consul_webhook_store", stdLog.Llongfile)
		stdLogger.SetOutput(log.NewStdlibAdapter(clientStore.options.logger))
		err = plan.RunWithClientAndLogger(clientStore.client, stdLogger)
		if err != nil {
			logging.Error(clientStore.options.logger).Log(logging.MessageKey(), "failed create plan", logging.ErrorKey(), err)
		}
	}()

	clientStore.plan = plan
	return clientStore
}

func (c *Client) handlePlanCallback(idx uint64, raw interface{}) {
	if raw == nil {
		return // ignore
	}
	_, ok := raw.(api.KVPairs)
	if !ok {
		return
	}
	if c.options.listener != nil {
		hooks, err := c.GetWebhook()
		if err == nil {
			c.options.listener.Update(hooks)
		} else {
			logging.Error(c.options.logger).Log(logging.MessageKey(), "failed to get webhooks ", logging.ErrorKey(), err)
		}
	}
}

func (c *Client) GetWebhook() ([]webhook.W, error) {
	hooks := []webhook.W{}
	kvPairs, _, err := c.client.KV().List(c.keyPrefix, &c.readOptions)
	if err != nil {
		return hooks, err
	}
	for _, kv := range kvPairs {
		hook := webhook.W{}
		err = json.Unmarshal(kv.Value, &hook)
		if err != nil {
			logging.Error(c.options.logger).Log(logging.MessageKey(), "failed to unmarshal webhook", logging.ErrorKey(), err)
			continue
		}
		hooks = append(hooks, hook)
	}
	return hooks, nil
}

func (c *Client) Push(w webhook.W) error {
	data, err := json.Marshal(&w)
	if err != nil {
		return err
	}
	_, err = c.client.KV().Put(&api.KVPair{
		Key:   c.keyPrefix + "/" + base64.RawURLEncoding.EncodeToString([]byte(w.ID())),
		Value: data,
	}, &c.writeOptions)
	return err
}

func (c *Client) Remove(id string) error {
	_, err := c.client.KV().Delete(c.keyPrefix+"/"+base64.RawURLEncoding.EncodeToString([]byte(id)), &c.writeOptions)
	return err
}

func (c *Client) Stop(context context.Context) {
	c.plan.Stop()
}
