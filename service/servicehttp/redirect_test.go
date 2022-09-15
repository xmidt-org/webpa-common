package servicehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

func testRedirectNoRequestURI(t *testing.T, expectedRedirectCode, actualRedirectCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		httpResponse = httptest.NewRecorder()
	)

	encoder := Redirect(actualRedirectCode)
	require.NotNil(encoder)

	err := encoder(
		logging.WithLogger(context.Background(), logging.NewTestLogger(nil, t)),
		httpResponse,
		"http://somewhere.com:8080",
	)

	assert.NoError(err)
	assert.Equal(expectedRedirectCode, httpResponse.Code)
	assert.Equal("http://somewhere.com:8080", httpResponse.Header().Get("Location"))
}

func testRedirectWithRequestURI(t *testing.T, expectedRedirectCode, actualRedirectCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		httpResponse = httptest.NewRecorder()
	)

	encoder := Redirect(actualRedirectCode)
	require.NotNil(encoder)

	err := encoder(
		context.WithValue(
			logging.WithLogger(context.Background(), logging.NewTestLogger(nil, t)),
			gokithttp.ContextKeyRequestURI,
			"/api/v2/device",
		),
		httpResponse,
		"http://somewhere.com:8080",
	)

	assert.NoError(err)
	assert.Equal(expectedRedirectCode, httpResponse.Code)
	assert.Equal("http://somewhere.com:8080/api/v2/device", httpResponse.Header().Get("Location"))
}

func TestRedirect(t *testing.T) {
	t.Run("NoRequestURI", func(t *testing.T) {
		testRedirectNoRequestURI(t, http.StatusTemporaryRedirect, 0)
		testRedirectNoRequestURI(t, http.StatusMovedPermanently, http.StatusMovedPermanently)
	})

	t.Run("WithRequestURI", func(t *testing.T) {
		testRedirectWithRequestURI(t, http.StatusTemporaryRedirect, 0)
		testRedirectWithRequestURI(t, http.StatusMovedPermanently, http.StatusMovedPermanently)
	})
}
