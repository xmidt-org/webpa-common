package xresolver

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: NewResolver(nil).DialContext,
		},
	}

	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	assert.NoError(err)
	assert.Equal(200, res.StatusCode)
}

type testCustomLookUp struct {
	nameHost map[string]string
	usedMap  bool
}

func (c *testCustomLookUp) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	item, found := c.nameHost[host]
	if found {
		records := make([]net.IPAddr, 1)
		records[0] = net.IPAddr{IP: net.ParseIP(item), Zone: ""}
		c.usedMap = true
		return records, nil
	}
	return []net.IPAddr{}, nil
}

func TestClientWithResolver(t *testing.T) {
	assert := assert.New(t)

	customhost := "custom.host.com"
	expectedBody := "Hello World\n"

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expectedBody)
	}))
	defer serverA.Close()

	hostA, portA, err := net.SplitHostPort(serverA.Listener.Addr().String())
	assert.NoError(err)

	r := NewResolver(nil)
	customLookUp := &testCustomLookUp{
		nameHost: map[string]string{customhost: hostA},
	}
	r.Add(customLookUp)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:       r.DialContext,
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequest("GET", "http://"+customhost+":"+portA, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	assert.NoError(err)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.NoError(err)

	assert.Equal(200, res.StatusCode)
	assert.Equal(expectedBody, string(body))
	assert.True(customLookUp.usedMap, "custom LookupIPAddr must be called")

	// Remove CustomLook up
	err = r.Remove(customLookUp)
	assert.NoError(err)

	req, err = http.NewRequest("GET", "http://"+customhost+":"+portA, nil)
	assert.NoError(err)

	res, err = client.Do(req)
	assert.Error(err)
}
