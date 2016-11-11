package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/billhathaway/consistentHash"
	"sort"
)

// Accessor provides access to services based around []byte keys.
// *consistentHash.ConsistentHash implements this interface.
type Accessor interface {
	Get([]byte) (string, error)
}

// AccessorFactory is a Factory Interface for creating service Accessors.
type AccessorFactory interface {
	// New creates an Accessor using a slice of endpoints
	New([]string) Accessor
}

// NewAccessorFactory uses a set of Options to produce an AccessorFactory
func NewAccessorFactory(o *Options) AccessorFactory {
	return &consistentHashFactory{
		logger:     o.logger(),
		vnodeCount: o.vnodeCount(),
	}
}

// consistentHashFactory creates consistentHash instances, which implement Accessor.
// This is the standard implementation of AccessorFactory.
type consistentHashFactory struct {
	logger     logging.Logger
	vnodeCount int
}

func (f *consistentHashFactory) New(endpoints []string) Accessor {
	hash := consistentHash.New()
	hash.SetVnodeCount(f.vnodeCount)

	if len(endpoints) > 0 {
		// make a sorted copy so that we add things in a consistent order
		sorted := make([]string, len(endpoints))
		copy(sorted, endpoints)
		sort.Strings(sorted)

		for _, hostAndPort := range sorted {
			f.logger.Debug("adding %s", hostAndPort)
			hash.Add(hostAndPort)
		}
	}

	return hash
}
