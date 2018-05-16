package fanout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testOptionsDefault(t *testing.T, o *Options) {
	assert := assert.New(t)
	assert.Empty(o.endpoints())
	assert.Equal("", o.authorization())
	assert.Equal(DefaultFanoutTimeout, o.fanoutTimeout())
	assert.Equal(DefaultClientTimeout, o.clientTimeout())
	assert.NotNil(o.transport())
	assert.Equal(DefaultConcurrency, o.concurrency())
	assert.Empty(o.redirectExcludeHeaders())
	assert.Zero(o.maxRedirects())
	assert.NotNil(o.checkRedirect())
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)

		o = Options{
			Endpoints:              []string{"localhost:1234"},
			Authorization:          "deadbeef",
			FanoutTimeout:          13 * time.Hour,
			ClientTimeout:          981 * time.Millisecond,
			Concurrency:            63482,
			RedirectExcludeHeaders: []string{"X-Test-1", "X-Test-2"},
			MaxRedirects:           17,
		}
	)

	assert.Equal([]string{"localhost:1234"}, o.endpoints())
	assert.Equal("deadbeef", o.authorization())
	assert.Equal(13*time.Hour, o.fanoutTimeout())
	assert.Equal(981*time.Millisecond, o.clientTimeout())
	assert.NotNil(o.transport())
	assert.Equal(63482, o.concurrency())
	assert.Equal([]string{"X-Test-1", "X-Test-2"}, o.redirectExcludeHeaders())
	assert.Equal(17, o.maxRedirects())
	assert.NotNil(o.checkRedirect())
}

func TestOptions(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		testOptionsDefault(t, nil)
	})

	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(t, new(Options))
	})

	t.Run("Custom", testOptionsCustom)
}

func TestNewTransactor(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(NewTransactor(Options{}))
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

		chain = NewChain(Options{})
	)

	decorated := chain.Then(handler)
	decorated.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.True(handlerCalled)
}
