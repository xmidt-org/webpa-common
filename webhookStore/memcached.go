package webhookStore

import (
	"context"
	"encoding/json"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/webhook"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemCachedConfig struct {
	Hosts []string
	TTL   time.Duration
}

type Memcached struct {
	options *storeConfig
	client  *memcache.Client
	config  MemCachedConfig
}

func (m *Memcached) GetWebhook() ([]webhook.W, error) {
	panic("implement me")
}

func (m *Memcached) Push(w webhook.W) error {
	data, err := json.Marshal(&w)
	if err != nil{
		return err
	}
	return m.client.Set(&memcache.Item{
		Key:        w.ID(),
		Value:      data,
		Flags:      0,
		Expiration: int32(m.config.TTL.Seconds()),
	})
}

func (m *Memcached) Remove(id string) error {
	return m.client.Delete(id)
}

func (m *Memcached) Stop(context context.Context) {
	m.client.st
}

func (m *Memcached) SetListener(listener Listener) error {
	panic("implement me")
}

func NewMemcachedClient(config MemCachedConfig, options ...Option) *Memcached {
	if len(config.Hosts) == 0 {
		return nil
	}
	m := &Memcached{
		options: &storeConfig{
			logger: logging.DefaultLogger(),
		},
	}
	for _, o := range options {
		o(m.options)
	}
	mc := memcache.New(config.Hosts...)
	m.client = mc
	return m
}
