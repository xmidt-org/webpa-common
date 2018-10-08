package consul

import (
	"fmt"
	"github.com/Comcast/webpa-common/service/monitor"
	"github.com/Comcast/webpa-common/xresolver"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConsulWatcher(t *testing.T) {
	assert := assert.New(t)

	customhost := "custom.host.com"
	customport := "8080"
	service := "custom"
	expectedBody := "Hello World\n"

	//customInstance := "custom.host-A.com"

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "a"+expectedBody)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "b"+expectedBody)
	}))
	defer serverB.Close()

	watcher := NewConsulWatcher(&Options{
		Watch: map[string]string{customhost: service},
	})

	// note: MonitorEvent is Listen interface in the monitor package
	watcher.MonitorEvent(monitor.Event{
		Key:       service + "[tag tagA]" + "{passingOnly=true}",
		Instances: []string{serverA.URL, serverB.URL},
	})

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: xresolver.NewResolver(nil, watcher).DialContext,
			// note: DisableKeepAlives is required so when we do the request again we don't reuse the same connection.
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequest("GET", "http://"+net.JoinHostPort(customhost, customport), nil)
	assert.NoError(err)

	res, err := client.Do(req)
	if assert.NoError(err) {

		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal("a"+expectedBody, string(body))
	}

	req, err = http.NewRequest("GET", "http://"+net.JoinHostPort(customhost, customport), nil)
	assert.NoError(err)

	res, err = client.Do(req)
	if assert.NoError(err) {

		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal("b"+expectedBody, string(body))
	}
}
