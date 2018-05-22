package fanout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testConfigurationDefault(t *testing.T, cfg *Configuration) {
	assert := assert.New(t)
	assert.Empty(cfg.endpoints())
	assert.Equal("", cfg.authorization())
	assert.Equal(DefaultFanoutTimeout, cfg.fanoutTimeout())
	assert.Equal(DefaultClientTimeout, cfg.clientTimeout())
	assert.NotNil(cfg.transport())
	assert.Equal(DefaultConcurrency, cfg.concurrency())
	assert.Empty(cfg.redirectExcludeHeaders())
	assert.Zero(cfg.maxRedirects())
	assert.NotNil(cfg.checkRedirect())
}

func testConfigurationCustom(t *testing.T) {
	var (
		assert = assert.New(t)

		cfg = Configuration{
			Endpoints:              []string{"localhost:1234"},
			Authorization:          "deadbeef",
			FanoutTimeout:          13 * time.Hour,
			ClientTimeout:          981 * time.Millisecond,
			Concurrency:            63482,
			RedirectExcludeHeaders: []string{"X-Test-1", "X-Test-2"},
			MaxRedirects:           17,
		}
	)

	assert.Equal([]string{"localhost:1234"}, cfg.endpoints())
	assert.Equal("deadbeef", cfg.authorization())
	assert.Equal(13*time.Hour, cfg.fanoutTimeout())
	assert.Equal(981*time.Millisecond, cfg.clientTimeout())
	assert.NotNil(cfg.transport())
	assert.Equal(63482, cfg.concurrency())
	assert.Equal([]string{"X-Test-1", "X-Test-2"}, cfg.redirectExcludeHeaders())
	assert.Equal(17, cfg.maxRedirects())
	assert.NotNil(cfg.checkRedirect())
}

func TestConfiguration(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		testConfigurationDefault(t, nil)
	})

	t.Run("Default", func(t *testing.T) {
		testConfigurationDefault(t, new(Configuration))
	})

	t.Run("Custom", testConfigurationCustom)
}

func TestNewTransactor(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewTransactor(Configuration{}))
}

func TestNewChain(t *testing.T) {
	var (
		assert = assert.New(t)

		handlerCalled = false
		handler       = http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			handlerCalled = true
			deadline, ok := request.Context().Deadline()
			assert.False(deadline.IsZero())
			assert.True(ok)
		})

		chain = NewChain(Configuration{})
	)

	decorated := chain.Then(handler)
	decorated.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.True(handlerCalled)
}
