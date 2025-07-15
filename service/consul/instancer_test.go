// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"strconv"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

// newServiceEntry creates a consul ServiceEntry with a service address
func newServiceEntry(serviceAddress string, port int, tags ...string) *api.ServiceEntry {
	return &api.ServiceEntry{
		Node: &api.Node{},
		Service: &api.AgentService{
			Address: serviceAddress,
			Port:    port,
			Tags:    tags,
		},
	}
}

// newServiceEntryNode creates a consul service entry with a node address
func newServiceEntryNode(nodeAddress string, port int, tags ...string) *api.ServiceEntry {
	return &api.ServiceEntry{
		Node: &api.Node{
			Address: nodeAddress,
		},
		Service: &api.AgentService{
			Port: port,
			Tags: tags,
		},
	}
}

func testFilterEntriesNil(t *testing.T) {
	var (
		assert   = assert.New(t)
		entries  []*api.ServiceEntry
		filtered = filterEntries(entries, nil)
	)

	assert.Len(filtered, 0)
}

func testFilterEntries(t *testing.T, unfiltered []*api.ServiceEntry, tags []string, expected []*api.ServiceEntry) {
	var (
		assert   = assert.New(t)
		filtered = filterEntries(unfiltered, tags)
	)

	assert.Equal(expected, filtered)
}

func TestFilterEntries(t *testing.T) {
	t.Run("Nil", testFilterEntriesNil)

	testData := []struct {
		unfiltered []*api.ServiceEntry
		tags       []string
		expected   []*api.ServiceEntry
	}{
		{
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 8080, "foo", "bar"),
				newServiceEntry("service2.com", 1234),
				newServiceEntryNode("node1.com", 9090, "bar"),
			},
			[]string{"foo"},
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 8080, "foo", "bar"),
			},
		},
		{
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 9567, "foo", "bar"),
				newServiceEntry("service2.com", 1111),
				newServiceEntryNode("node1.com", 2222, "bar"),
				newServiceEntryNode("node2.com", 671, "bar", "foo"),
				newServiceEntryNode("node3.com", 772, "bar", "foo", "moo"),
				newServiceEntry("service3.com", 12560, "moo", "foo", "bar"),
			},
			[]string{"foo", "bar"},
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 9567, "foo", "bar"),
				newServiceEntryNode("node2.com", 671, "bar", "foo"),
				newServiceEntryNode("node3.com", 772, "bar", "foo", "moo"),
				newServiceEntry("service3.com", 12560, "moo", "foo", "bar"),
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			testFilterEntries(t, record.unfiltered, record.tags, record.expected)
		})
	}
}

func TestMakeInstances(t *testing.T) {
	testData := []struct {
		entries  []*api.ServiceEntry
		expected []string
	}{
		{
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 4343, "foo", "bar"),
				newServiceEntry("service2.com", 1717),
				newServiceEntryNode("node1.com", 901, "bar"),
			},
			[]string{"service1.com:4343", "service2.com:1717", "node1.com:901"},
		},
		{
			[]*api.ServiceEntry{
				newServiceEntry("service1.com", 8080, "foo", "bar"),
				newServiceEntry("service2.com", 9090),
				newServiceEntryNode("node1.com", 16721, "bar"),
				newServiceEntryNode("node2.com", 6, "bar", "foo"),
				newServiceEntryNode("node3.com", 916, "bar", "foo", "moo"),
				newServiceEntry("service3.com", 99, "moo", "foo", "bar"),
			},
			[]string{"service1.com:8080", "service2.com:9090", "node1.com:16721", "node2.com:6", "node3.com:916", "service3.com:99"},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(record.expected, makeInstances(record.entries))
		})
	}
}
