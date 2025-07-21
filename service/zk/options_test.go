// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package zk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testRegistrationDefault(t *testing.T, r Registration) {
	assert := assert.New(t)

	assert.Equal(DefaultServiceName, r.name())
	assert.Equal(DefaultPath, r.path())
	assert.Equal(DefaultAddress, r.address())
	assert.Equal(DefaultPort, r.port())
	assert.Equal(DefaultScheme, r.scheme())
}

func testRegistrationCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		r      = Registration{
			Name:    "testService",
			Path:    "/testy/test/test",
			Address: "funzo.net",
			Port:    1234,
			Scheme:  "ftp",
		}
	)

	assert.Equal("testService", r.name())
	assert.Equal("/testy/test/test", r.path())
	assert.Equal("funzo.net", r.address())
	assert.Equal(1234, r.port())
	assert.Equal("ftp", r.scheme())
}

func TestRegistration(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testRegistrationDefault(t, Registration{})
	})

	t.Run("Custom", testRegistrationCustom)
}

func testClientDefault(t *testing.T, c *Client) {
	assert := assert.New(t)

	assert.Equal([]string{DefaultServer}, c.servers())
	assert.Equal(DefaultConnectTimeout, c.connectTimeout())
	assert.Equal(DefaultSessionTimeout, c.sessionTimeout())
}

func testClientCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = Client{
			Connection:     "localhost:1234",
			Servers:        []string{"somewhere.com:8888"},
			ConnectTimeout: 13 * time.Hour,
			SessionTimeout: 1239 * time.Minute,
		}
	)

	assert.Equal([]string{"localhost:1234", "somewhere.com:8888"}, c.servers())
	assert.Equal(13*time.Hour, c.connectTimeout())
	assert.Equal(1239*time.Minute, c.sessionTimeout())
}

func TestClient(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testClientDefault(t, nil)
		testClientDefault(t, new(Client))
	})

	t.Run("Custom", testClientCustom)
}

func testOptionsDefault(t *testing.T, o *Options) {
	var (
		assert = assert.New(t)
		c      = o.client()
	)

	assert.Equal([]string{DefaultServer}, c.servers())
	assert.Equal(DefaultConnectTimeout, c.connectTimeout())
	assert.Equal(DefaultSessionTimeout, c.sessionTimeout())
	assert.Len(o.registrations(), 0)
	assert.Len(o.watches(), 0)
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		o      = Options{
			Client: Client{
				Connection:     "localhost:1234",
				Servers:        []string{"somewhere.com:8888"},
				ConnectTimeout: 13 * time.Hour,
				SessionTimeout: 1239 * time.Minute,
			},
			Registrations: []Registration{
				Registration{
					Name:    "testService",
					Path:    "/testy/test/test",
					Address: "funzo.net",
					Port:    1234,
					Scheme:  "ftp",
				},
			},
			Watches: []string{"/testy/test/test"},
		}

		c = o.client()
	)

	assert.Equal([]string{"localhost:1234", "somewhere.com:8888"}, c.servers())
	assert.Equal(13*time.Hour, c.connectTimeout())
	assert.Equal(1239*time.Minute, c.sessionTimeout())
	assert.Equal(
		[]Registration{
			Registration{
				Name:    "testService",
				Path:    "/testy/test/test",
				Address: "funzo.net",
				Port:    1234,
				Scheme:  "ftp",
			},
		},
		o.registrations(),
	)

	assert.Equal([]string{"/testy/test/test"}, o.watches())
}

func TestOptions(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(t, nil)
		testOptionsDefault(t, new(Options))
	})

	t.Run("Custom", testOptionsCustom)
}
