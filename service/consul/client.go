// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	gokitconsul "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
)

// Client extends the go-kit consul Client interface with behaviors specific to XMiDT
type Client interface {
	gokitconsul.Client

	// Datacenters returns the known datacenters from the catalog
	Datacenters() ([]string, error)
}

// NewClient constructs a Client object which wraps the given hashicorp consul client.
// This factory function is the analog to go-kit's sd/consul.NewClient function.
func NewClient(c *api.Client) Client {
	return client{
		gokitconsul.NewClient(c),
		c,
	}
}

// client implements go-kit's consul Client interface and extends it to the local Client interface
type client struct {
	gokitconsul.Client
	c *api.Client
}

func (c client) Datacenters() ([]string, error) {
	return c.c.Catalog().Datacenters()
}
