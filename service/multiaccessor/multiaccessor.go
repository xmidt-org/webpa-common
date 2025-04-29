// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"errors"
)

// MultiAccessor defines the hashing behavior for multi hashing service discover
type MultiAccessor interface {
	// Get returns a list server nodes associated with a particular key.
	Get(key []byte) ([]string, error)
	// GetNormMap returns a dict of server node normalization dicts.
	GetNormMap() (m map[HasherType]map[string]string)
}

// New returns a MultiAccessor.
func New(opts ...option) MultiAccessor {
	hs := make(multiHasher)
	options(opts).Apply(hs)

	return hs
}

// multiHasher implements the `MultiAccessor` interface.
type multiHasher map[HasherType]hasher

// Add adds a server node to the multiHasher.
func (hs multiHasher) Add(s string) {
	for _, h := range hs {
		h.Add(s)
	}
}

// Get returns a list server nodes associated with a particular key.
func (hs multiHasher) Get(key []byte) (servers []string, errs error) {
	for _, h := range hs {
		s, err := h.Get(key)
		servers = append(servers, s)
		errs = errors.Join(errs, err)
	}

	return servers, errs
}

// GetNormMap returns a dict of server node normalization dicts.
func (hs multiHasher) GetNormMap() map[HasherType]map[string]string {
	m := make(map[HasherType]map[string]string)
	for n, h := range hs {
		m[n] = h.GetNormMap()
	}

	return m
}
