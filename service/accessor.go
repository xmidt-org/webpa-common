package service

import (
	"sort"
	"strings"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/billhathaway/consistentHash"
)

const (
	DefaultVNodeCount = 211
)

// InstancesFilter represents a function which can preprocess slices of instances from the
// service discovery subsystem.
type InstancesFilter func([]string) []string

// DefaultInstancesFilter removes blank nodes and sorts the remaining nodes so that
// there is a consistent ordering.
func DefaultInstancesFilter(original []string) []string {
	filtered := make([]string, 0, len(original))

	for _, o := range original {
		f := strings.TrimSpace(o)
		if len(f) > 0 {
			filtered = append(filtered, f)
		}
	}

	sort.Strings(filtered)
	return filtered
}

// InstrumentFilter applies a Setter (gauge) that records the count of instances from the next filter.
func InstrumentFilter(count xmetrics.Setter, next InstancesFilter) InstancesFilter {
	return func(raw []string) []string {
		filtered := next(raw)
		count.Set(float64(len(filtered)))
		return filtered
	}
}

// AccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an Accessor.
//
// A Subscription will use an InstancesFilter prior to invoking this factory.
type AccessorFactory func([]string) Accessor

// ConsistentAccessorFactory produces a factory which uses consistent hashing
// of server nodes.
func ConsistentAccessorFactory(vnodeCount int) AccessorFactory {
	if vnodeCount < 1 {
		vnodeCount = DefaultVNodeCount
	}

	return func(instances []string) Accessor {
		hasher := consistentHash.New()
		hasher.SetVnodeCount(vnodeCount)
		for _, i := range instances {
			hasher.Add(i)
		}

		return hasher
	}
}

// Accessor holds a hash of server nodes.
type Accessor interface {
	// Get fetches the server node associated with a particular key.
	Get(key []byte) (string, error)
}
