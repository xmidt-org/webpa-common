package xhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/logging"
)

func testRedirectPolicyDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		p       = RedirectPolicy{}
	)

	assert.Equal(DefaultMaxRedirects, p.maxRedirects())

	f := p.headerFilter()
	require.NotNil(f)
	assert.True(f("something"))
}

func testRedirectPolicyCustom(t *testing.T) {
	var (
		assert         = assert.New(t)
		require        = require.New(t)
		expectedLogger = logging.NewTestLogger(nil, t)

		p = RedirectPolicy{
			Logger:         expectedLogger,
			MaxRedirects:   7,
			ExcludeHeaders: []string{"content-type"},
		}
	)

	assert.Equal(7, p.maxRedirects())

	f := p.headerFilter()
	require.NotNil(f)
	assert.True(f("Accept"))
	assert.False(f("Content-Type"))
}

func TestRedirectPolicy(t *testing.T) {
	t.Run("Default", testRedirectPolicyDefault)
	t.Run("Custom", testRedirectPolicyCustom)
}

func testCheckRedirectMaxRedirects(t *testing.T) {
	var (
		assert = assert.New(t)

		via = []*http.Request{
			httptest.NewRequest("GET", "/first", nil),
			httptest.NewRequest("GET", "/second", nil),
		}

		checkRedirect = CheckRedirect(
			RedirectPolicy{
				MaxRedirects: 2,
			},
		)
	)

	assert.Error(checkRedirect(httptest.NewRequest("GET", "/", nil), via))
}

func testCheckRedirectCopyHeaders(t *testing.T) {
	var (
		assert = assert.New(t)

		checkRedirect = CheckRedirect(RedirectPolicy{
			Logger:         logging.NewTestLogger(nil, t),
			ExcludeHeaders: []string{"content-type", "X-Supar-Sekrit"},
		})

		r   = httptest.NewRequest("GET", "/", nil)
		via = []*http.Request{
			httptest.NewRequest("GET", "/", nil),
		}
	)

	via[len(via)-1].Header.Set("Content-Type", "text/plain")
	via[len(via)-1].Header.Add("x-supar-sekrit", "don't reveal me, bro!")

	via[len(via)-1].Header.Set("X-Something", "value")
	via[len(via)-1].Header.Add("X-Something-Else", "value1")
	via[len(via)-1].Header.Add("X-Something-Else", "value2")

	checkRedirect(r, via)
	assert.Equal("value", r.Header.Get("X-Something"))
	assert.Equal([]string{"value1", "value2"}, r.Header["X-Something-Else"])
	assert.Equal("", r.Header.Get("Content-Type"))
	assert.Equal("", r.Header.Get("X-Supar-Sekrit"))
}

func TestCheckRedirect(t *testing.T) {
	t.Run("MaxRedirects", testCheckRedirectMaxRedirects)
	t.Run("CopyHeaders", testCheckRedirectCopyHeaders)
}
