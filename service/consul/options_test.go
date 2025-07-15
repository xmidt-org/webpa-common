// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptionsDefault(t *testing.T, o *Options) {
	assert := assert.New(t)

	assert.NotNil(o.config())
	assert.False(o.disableGenerateID())
	assert.Len(o.registrations(), 0)
	assert.Len(o.watches(), 0)
}

func testOptionsCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		o = Options{
			Client: &api.Config{
				Address: "somewhere.com",
				Scheme:  "ftp",
			},

			DisableGenerateID: true,

			Registrations: []api.AgentServiceRegistration{
				api.AgentServiceRegistration{
					ID:   "foo",
					Name: "bar",
				},
			},

			Watches: []Watch{
				Watch{
					Service:     "moo",
					Tags:        []string{"a", "b"},
					PassingOnly: true,
				},
			},
		}
	)

	c := o.config()
	require.NotNil(c)
	assert.Equal("somewhere.com", c.Address)
	assert.Equal("ftp", c.Scheme)

	assert.True(o.disableGenerateID())

	assert.Equal(
		[]api.AgentServiceRegistration{
			api.AgentServiceRegistration{
				ID:   "foo",
				Name: "bar",
			},
		},
		o.registrations(),
	)

	assert.Equal(
		[]Watch{
			Watch{
				Service:     "moo",
				Tags:        []string{"a", "b"},
				PassingOnly: true,
			},
		},
		o.watches(),
	)
}

func TestOptions(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		testOptionsDefault(t, nil)
		testOptionsDefault(t, new(Options))
	})

	t.Run("Custom", testOptionsCustom)
}
