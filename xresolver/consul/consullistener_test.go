package consul

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/service/monitor"
	"github.com/xmidt-org/webpa-common/v2/xresolver"
)

func TestConsulWatcher(t *testing.T) {
	assert := assert.New(t)

	customhost := "custom.host.com"
	customport := "8080"
	service := "custom"
	expectedBody := "Hello World\n"
	fallBackURL := "http://" + net.JoinHostPort(customhost, customport)

	// customInstance := "custom.host-A.com"

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "a"+expectedBody)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "b"+expectedBody)
	}))
	defer serverB.Close()

	watcher := NewConsulWatcher(Options{
		Watch: map[string]string{fallBackURL: service},
	})

	// note: MonitorEvent is Listen interface in the monitor package
	watcher.MonitorEvent(monitor.Event{
		Key:       service + "[tag tagA]" + "{passingOnly=true}",
		Instances: []string{serverA.URL, serverB.URL},
	})

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: xresolver.NewResolver(xresolver.DefaultDialer, logging.NewTestLogger(nil, t), watcher).DialContext,
			// note: DisableKeepAlives is required so when we do the request again we don't reuse the same connection.
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequest("GET", fallBackURL, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	if assert.NoError(err) {

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal("a"+expectedBody, string(body))
	}

	req, err = http.NewRequest("GET", fallBackURL, nil)
	assert.NoError(err)

	res, err = client.Do(req)
	if assert.NoError(err) {

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal("b"+expectedBody, string(body))
	}
}
