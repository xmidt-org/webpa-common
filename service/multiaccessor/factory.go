// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

const DefaultVnodeCount = 211

// MultiAccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an MultiAccessor.
type MultiAccessorFactory func([]string) MultiAccessor

// NewMultiAccessorFactory produces a factory which uses consistent hashing
// of server nodes.  The returned factory does not modify instances passed to it.
// Instances are hashed as is.
func NewMultiAccessorFactory(b Builder, c int) MultiAccessorFactory {
	// Buidler always uses DefaultVnodeCount as default
	return func(instances []string) MultiAccessor {
		return b.Build(VnodeCount(c), Instances(instances))
	}
}

// DefaultMultiAccessorFactory is the default strategy for creating MultiAccessor based on a set of instances.
// This default creates consistent hashing accessors.
func DefaultMultiAccessorFactory(instances []string) MultiAccessor {
	return Builder{}.Build(Instances(instances))
}
