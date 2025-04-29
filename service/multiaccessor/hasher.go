// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"github.com/billhathaway/consistentHash"
)

// hasher defines hashing behavior for individual hashers used by MultiAccessor
type hasher interface {
	// Get returns the server node associated with a particular normalized key.
	Get(key []byte) (string, error)
	// Add adds a server node.
	Add(s string)
	// SetVnodeCount sets the number of vnodes that will be added for every server.
	SetVnodeCount(int) error
	// GetNormMap returns the server node normalization dict.
	// Note, this is helpful to debug service discovery issues.
	GetNormMap() map[string]string
}

// NewHasher returns a hasher.
func NewHasher(n normalizer) hasher {
	return normHasher{
		hasher:  consistentHash.New(),
		norm:    n,
		normMap: map[string]string{},
	}
}

// normHasher implements the `hasher` interface while allowing configurable
// hash normalization.
type normHasher struct {
	hasher  *consistentHash.ConsistentHash
	norm    normalizer
	normMap map[string]string
}

// Add adds a server url to the consistentHash.
// Note, Add expects a valid url with a schema and hostname
func (h normHasher) Add(url string) {
	k := h.norm(url)
	h.hasher.Add(k)
	h.normMap[k] = url
}

// Get returns a server node associated with a particular key.
func (h normHasher) Get(key []byte) (string, error) {
	host, err := h.hasher.Get(key)
	if err != nil {
		return "", err
	}

	return h.normMap[host], nil
}

// SetVnodeCount sets the number of vnodes that will be added for every server.
func (h normHasher) SetVnodeCount(c int) error {
	return h.hasher.SetVnodeCount(c)
}

// GetNormMap returns a dict of server node normalization dicts.
func (h normHasher) GetNormMap() map[string]string {
	return h.normMap
}
