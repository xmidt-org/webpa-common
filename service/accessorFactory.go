package service

import "github.com/billhathaway/consistentHash"

const DefaultVnodeCount = 211

// AccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an Accessor.
type AccessorFactory func([]string) Accessor

func newConsistentAccessor(vnodeCount int, instances []string) Accessor {
	if len(instances) == 0 {
		return emptyAccessor{}
	}

	hasher := consistentHash.New()
	hasher.SetVnodeCount(vnodeCount)
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

// DefaultAccessorFactory is the default strategy for creating Accessors based on a set of instances.
// This default creates consistent hashing accessors.
func DefaultAccessorFactory(instances []string) Accessor {
	return newConsistentAccessor(DefaultVnodeCount, instances)
}
