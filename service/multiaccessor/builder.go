// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

// Builder is a configuration struct used to configure multi hashing service discover.
// Note, HostNameHasher is used as the default hasher if Builder is empty.
type Builder struct {
	MultiAccessor []HasherConfig `json:"hashers"`
}

// Build translates the configuration into a MultiAccessor.
func (b Builder) Build(opts ...option) MultiAccessor {
	var hopts options
	for _, c := range b.MultiAccessor {
		if c.Disable {
			continue
		}

		hopts = append(hopts, Hasher(c.Type))
	}

	if len(hopts) == 0 {
		hopts = append(hopts, Hasher(defaultHasherType))
	}

	defaults := options{
		VnodeCount(DefaultVnodeCount),
	}
	opts = append(defaults, opts...)

	// Apply hasher options first
	return New(append(hopts, opts...))
}

type HasherConfig struct {
	// Type assigns the hasher type.
	Type HasherType `json:"type"`
	// Disable determines whether the hasher is active (`diable` is `false`)
	// or inactive (`disable` is `true`).
	// Default is `false`.
	Disable bool `json:"disable"`
}
