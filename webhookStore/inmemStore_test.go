package webhookStore

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"testing"
	"time"
)

func TestImplementsInterfaces(t *testing.T) {
	var (
		inmem interface{}
	)
	assert := assert.New(t)
	inmem = CreateInMemStore(InMemConfig{TTL: time.Second})
	_, ok := inmem.(Pusher)
	assert.True(ok, "not an webhook Push")
	_, ok = inmem.(Listener)
	assert.True(ok, "not an webhook Listener")
	_, ok = inmem.(Reader)
	assert.True(ok, "not a webhook Reader")
}

var (
	neatWebhook = webhook.W{
		Config: struct {
			URL             string   `json:"url"`
			ContentType     string   `json:"content_type"`
			Secret          string   `json:"secret,omitempty"`
			AlternativeURLs []string `json:"alt_urls,omitempty"`
		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "idontknow"},
		Events: []string{".*"},
	}
	neatWebhookWithDifferentSecret = webhook.W{
		Config: struct {
			URL             string   `json:"url"`
			ContentType     string   `json:"content_type"`
			Secret          string   `json:"secret,omitempty"`
			AlternativeURLs []string `json:"alt_urls,omitempty"`
		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "ohnowiknow"},
		Events: []string{".*"},
	}
)

func TestInMemWithNoOptions(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := CreateInMemStore(InMemConfig{
		TTL:           time.Second,
		CheckInterval: time.Millisecond * 10,
	})
	require.NotNil(client)

	// test push
	err := client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err := client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)

	// test remove
	err = client.Remove(neatWebhook.ID())
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{}, hooks)

	// test ttl
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)
	time.Sleep(time.Second * 2)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{}, hooks)

	// test update
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)
	err = client.Push(neatWebhookWithDifferentSecret)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhookWithDifferentSecret}, hooks)
}

type hookListener struct {
	hooks  []webhook.W
	listen ListenerFunc
}

func (listner *hookListener) Update(hooks []webhook.W) {
	listner.hooks = hooks
}

func TestInMemWithListener(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	listener := &hookListener{}

	client := CreateInMemStore(InMemConfig{
		TTL:           time.Second,
		CheckInterval: time.Millisecond * 10,
	}, WithListener(listener))
	require.NotNil(client)

	// test push
	err := client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err := client.GetWebhook()
	assert.NoError(err)
	time.Sleep(time.Millisecond)
	assert.Equal(hooks, listener.hooks)

	// test remove
	err = client.Remove(neatWebhook.ID())
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	time.Sleep(time.Millisecond)
	assert.Equal(hooks, listener.hooks)

	// test ttl
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal(hooks, listener.hooks)
	time.Sleep(time.Second * 2)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	time.Sleep(time.Millisecond)
	assert.Equal(hooks, listener.hooks)

	// test update
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	time.Sleep(time.Millisecond)
	assert.Equal(hooks, listener.hooks)
	err = client.Push(neatWebhookWithDifferentSecret)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	time.Sleep(time.Millisecond)
	assert.Equal(hooks, listener.hooks)
}

type hookStorage struct {
	hooks   map[string]webhook.W
	options *storeConfig
}

func (h *hookStorage) Push(w webhook.W) error {
	h.hooks[w.ID()] = w
	if h.options.listener != nil {
		hooks, _ := h.GetWebhook()
		h.options.listener.Update(hooks)
	}
	return nil
}

func (h *hookStorage) Remove(id string) error {
	delete(h.hooks, id)
	if h.options.listener != nil {
		hooks, _ := h.GetWebhook()
		h.options.listener.Update(hooks)
	}
	return nil
}

func (h *hookStorage) Stop(context context.Context) {
}

func (h *hookStorage) GetWebhook() ([]webhook.W, error) {
	data := []webhook.W{}
	for _, value := range h.hooks {
		data = append(data, value)
	}
	return data, nil
}

func TestInMemWithBackend(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := CreateInMemStore(InMemConfig{
		TTL:           time.Second,
		CheckInterval: time.Millisecond * 10,
	}, WithStorageListener(func(options ...Option) Pusher {
		storage := &hookStorage{
			hooks: map[string]webhook.W{},
		}
		storage.options = &storeConfig{
			logger: logging.NewTestLogger(nil, t),
			self:   storage,
		}
		for _, o := range options {
			o(storage.options)
		}
		return storage
	}))
	require.NotNil(client)

	// test push
	err := client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err := client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)

	// test remove
	err = client.Remove(neatWebhook.ID())
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{}, hooks)

	// test ttl
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)
	time.Sleep(time.Second * 2)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{}, hooks)

	// test update
	err = client.Push(neatWebhook)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhook}, hooks)
	err = client.Push(neatWebhookWithDifferentSecret)
	assert.NoError(err)
	hooks, err = client.GetWebhook()
	assert.NoError(err)
	assert.Equal([]webhook.W{neatWebhookWithDifferentSecret}, hooks)
}
