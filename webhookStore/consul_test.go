package webhookStore

import (
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
		Client:       client,
		WriteOptions: api.WriteOptions{},
		Prefix:       "testing",
	})
	require.NotEmpty(consulStore)
	_, ok := consulStore.(Pusher)
	assert.True(ok, "not an webhook Pusher")
	_, ok = consulStore.(Reader)
	assert.True(ok, "not a webhook Reader")
}

// func TestConsulIntegration(t *testing.T) {
// 	assert := assert.New(t)
// 	require := require.New(t)
// 	client, err := api.NewClient(&api.Config{})
// 	require.NoError(err)
// 	dataChan := make(chan struct{}, 1)
//
// 	var listner ListnerFunc
// 	var resultingHooks []webhook.W
// 	listner = func(hooks []webhook.W) {
// 		resultingHooks = hooks
// 		dataChan <- struct{}{}
// 	}
//
// 	consulStore := CreateConsulStore(ConsulConfig{
// 		Client:       client,
// 		WriteOptions: api.WriteOptions{},
// 		Prefix:       "testing",
// 	}, WithListener(listner), WithLogger(logging.NewTestLogger(nil, t)))
// 	require.NotNil(consulStore)
// 	<-dataChan
//
//
// 	expectedWebhook := webhook.W{
// 		Config: struct {
// 			URL             string   `json:"url"`
// 			ContentType     string   `json:"content_type"`
// 			Secret          string   `json:"secret,omitempty"`
// 			AlternativeURLs []string `json:"alt_urls,omitempty"`
// 		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "idontknow"},
// 		Events: []string{".*"},
// 	}
// 	err = consulStore.Push(expectedWebhook)
// 	assert.NoError(err)
// 	<-dataChan
// 	assert.Equal([]webhook.W{expectedWebhook}, resultingHooks)
// 	expectedWebhook = webhook.W{
// 		Config: struct {
// 			URL             string   `json:"url"`
// 			ContentType     string   `json:"content_type"`
// 			Secret          string   `json:"secret,omitempty"`
// 			AlternativeURLs []string `json:"alt_urls,omitempty"`
// 		}{URL: "http://localhost/events?neat", ContentType: "json", Secret: "ohnowiknow"},
// 		Events: []string{".*", "device-status"},
// 	}
//
// 	err = consulStore.Push(expectedWebhook)
// 	assert.NoError(err)
// 	<-dataChan
// 	assert.Equal([]webhook.W{expectedWebhook}, resultingHooks)
//
// 	err = consulStore.Remove("http://localhost/events?neat")
// 	assert.NoError(err)
//
// 	tempHooks, err := consulStore.GetWebhook()
// 	assert.NoError(err)
// 	assert.Empty(tempHooks)
// }
