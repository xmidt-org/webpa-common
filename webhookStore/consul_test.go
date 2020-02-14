package webhookStore

import (
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"testing"
	"time"
)

func TestConsulInterfaces(t *testing.T) {
	var (
		consulStore interface{}
	)
	assert := assert.New(t)
	require := require.New(t)
	client, err := api.NewClient(&api.Config{})
	require.NoError(err)
	consulStore = CreateConsulStore(ConsulConfig{
		Client: client,
		Prefix: "testing",
	})
	require.NotEmpty(consulStore)
	_, ok := consulStore.(Pusher)
	assert.True(ok, "not an webhook Pusher")
	_, ok = consulStore.(Reader)
	assert.True(ok, "not a webhook Reader")
}

func TestInMemWithConsul(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := logging.NewTestLogger(nil, t)
	client, err := api.NewClient(&api.Config{})
	require.NoError(err)
	consulStore := CreateConsulStore(ConsulConfig{
		Client: client,
		Prefix: "testing",
	}, WithLogger(logger))
	webhookStore := CreateInMemStore(InMemConfig{}, WithLogger(logger), WithStorage(consulStore))
	consulStore.SetListener(webhookStore)
	assert.NotNil(webhookStore)
}

func TestConsulIntegration(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := logging.NewTestLogger(nil, t)
	client, err := api.NewClient(&api.Config{})
	require.NoError(err)

	var listener ListenerFunc
	var resultingHooks []webhook.W
	listener = func(hooks []webhook.W) {
		resultingHooks = hooks
	}

	consulStore := CreateConsulStore(ConsulConfig{
		Client: client,
		Prefix: "testing",
	}, WithLogger(logger))
	webhookStore := CreateInMemStore(InMemConfig{}, WithLogger(logger), WithStorage(consulStore), WithListener(listener))
	consulStore.SetListener(webhookStore)
	require.NotNil(webhookStore)

	expectedWebhook := webhook.W{
		Config: struct {
			URL             string   `json:"url"`
			ContentType     string   `json:"content_type"`
			Secret          string   `json:"secret,omitempty"`
			AlternativeURLs []string `json:"alt_urls,omitempty"`
		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "idontknow"},
		Events: []string{".*"},
	}
	err = webhookStore.Push(expectedWebhook)
	assert.NoError(err)
	time.Sleep(time.Second)
	assert.Equal([]webhook.W{expectedWebhook}, resultingHooks)
	expectedWebhook = webhook.W{
		Config: struct {
			URL             string   `json:"url"`
			ContentType     string   `json:"content_type"`
			Secret          string   `json:"secret,omitempty"`
			AlternativeURLs []string `json:"alt_urls,omitempty"`
		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "ohnowiknow"},
		Events: []string{".*", "device-status"},
	}

	err = webhookStore.Push(expectedWebhook)
	assert.NoError(err)
	time.Sleep(time.Second)
	assert.Equal([]webhook.W{expectedWebhook}, resultingHooks)

	err = webhookStore.Remove("http://localhost/events?neat")
	assert.NoError(err)

	tempHooks, err := webhookStore.GetWebhook()
	assert.NoError(err)
	assert.Empty(tempHooks)
}
