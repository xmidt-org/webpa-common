package webhookStore

import (
	"github.com/stretchr/testify/assert"
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
