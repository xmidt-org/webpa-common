package accessor

import (
	"fmt"
	"net/url"

	"github.com/billhathaway/consistentHash"
	"github.com/xmidt-org/webpa-common/v2/xhttp/gate"
)

const DefaultVnodeCount = 211

// AccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an Accessor.
type AccessorFactory func([]string) Accessor

type hostHasher struct {
	hasher    *consistentHash.ConsistentHash
	hostToURL map[string]string
}

// Add adds a server url to the consistentHash.
// Note, Add expects a valid url with a schema and hostname
func (h *hostHasher) Add(s string) {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse url `%s`: %s", s, err))
	}
	if u.Scheme == "" || u.Host == "" {
		panic(fmt.Sprintf("expected a schema and host: `%s`", s))
	}

	h.hasher.Add(u.Hostname())
	h.hostToURL[u.Hostname()] = u.String()
}

// Get fetches the server url associated with a particular key.
func (h *hostHasher) Get(key []byte) (string, error) {
	host, err := h.hasher.Get(key)
	if err != nil {
		return "", err
	}

	return h.hostToURL[host], nil
}

func newHostHasher(vnodeCount int) *hostHasher {
	hasher := consistentHash.New()
	hasher.SetVnodeCount(vnodeCount)
	return &hostHasher{
		hasher:    hasher,
		hostToURL: map[string]string{},
	}
}

func newConsistentAccessor(vnodeCount int, instances []string) Accessor {
	if len(instances) == 0 {
		return emptyAccessor{}
	}

	hasher := newHostHasher(vnodeCount)
	for _, i := range instances {
		hasher.Add(i)
	}

	return hasher
}

// NewConsistentAccessorFactory produces a factory which uses consistent hashing
// of server nodes.  The returned factory does not modify instances passed to it.
// Instances are hashed as is.
//
// If vnodeCount is nonpositive or equal to DefaultVnodeCount, the returned factory
// is the DefaultAccessorFactory.
func NewConsistentAccessorFactory(vnodeCount int) AccessorFactory {
	if vnodeCount < 1 || vnodeCount == DefaultVnodeCount {
		return DefaultAccessorFactory
	}

	return func(instances []string) Accessor {
		return newConsistentAccessor(vnodeCount, instances)
	}
}

// NewConsistentAccessorFactoryWithGate produces a factory which uses consistent hashing
// of server nodes with the gate feature.  The returned factory does not modify instances passed to it.
// Instances are hashed as is. If the gate is closed an error saying the gate is closed will be returned
func NewConsistentAccessorFactoryWithGate(vnodeCount int, g gate.Interface) AccessorFactory {
	if vnodeCount < 1 {
		return func(instances []string) Accessor {
			return GateAccessor(g, newConsistentAccessor(DefaultVnodeCount, instances))
		}
	}

	return func(instances []string) Accessor {
		return GateAccessor(g, newConsistentAccessor(vnodeCount, instances))
	}
}

// DefaultAccessorFactory is the default strategy for creating Accessors based on a set of instances.
// This default creates consistent hashing accessors.
func DefaultAccessorFactory(instances []string) Accessor {
	return newConsistentAccessor(DefaultVnodeCount, instances)
}
