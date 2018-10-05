package xresolver

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
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
	nameHost map[string]Route
	usedMap  bool
}

func (c *testCustomLookUp) LookupRoutes(ctx context.Context, host string) ([]Route, error) {
	if route, found := c.nameHost[host]; found {
		records := make([]Route, 1)
		records[0] = route
		c.usedMap = true
		return records, nil
	}
	return []Route{}, errors.New("no routes found")
}

func TestClientWithResolver(t *testing.T) {
	assert := assert.New(t)

	customhost := "custom.host.com"
	customport := "8080"
	expectedBody := "Hello World\n"

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expectedBody)
	}))
	defer serverA.Close()

	route, err := CreateRoute(serverA.URL)
	assert.NoError(err)

	customLookUp := &testCustomLookUp{
		nameHost: map[string]Route{customhost: *route},
	}
	r := NewResolver(nil, customLookUp)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:       r.DialContext,
			DisableKeepAlives: true,
		},
	}

	req, err := http.NewRequest("GET", "http://"+customhost+":"+customport, nil)
	assert.NoError(err)

	res, err := client.Do(req)
	if assert.NoError(err) {
		body, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(err)

		assert.Equal(200, res.StatusCode)
		assert.Equal(expectedBody, string(body))
		assert.True(customLookUp.usedMap, "custom LookupIPAddr must be called")
	}

	// Remove CustomLook up
	err = r.Remove(customLookUp)
	assert.NoError(err)

	req, err = http.NewRequest("GET", "http://"+customhost+":"+customport, nil)
	assert.NoError(err)

	res, err = client.Do(req)
	assert.Error(err)
}
