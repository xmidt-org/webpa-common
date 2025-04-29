// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type HasherType int

const (
	UnknownType HasherType = iota
	RawURLType
	HostnameType
	lastType
)

var ErrHasherTypeInvalid = errors.New("hasher type is invalid")

var (
	HasherTypeUnmarshal = map[string]HasherType{
		"unknown":  UnknownType,
		"raw_url":  RawURLType,
		"hostname": HostnameType,
	}
	HasherTypeMarshal = map[HasherType]string{
		UnknownType:  "unknown",
		RawURLType:   "raw_url",
		HostnameType: "hostname",
	}
)

// IsEmpty returns true if the value is UnknownType (the default),
// otherwise false is returned.
func (ht HasherType) IsEmpty() bool {
	return ht == UnknownType
}

// IsValid returns true if the value is valid,
// otherwise false is returned.
func (ht HasherType) IsValid() bool {
	return UnknownType < ht && ht < lastType
}

// String returns a human-readable string representation for an existing HasherType,
// otherwise String returns the `unknown` string value.
func (ht HasherType) String() string {
	if value, ok := HasherTypeMarshal[ht]; ok {
		return value
	}

	return HasherTypeMarshal[UnknownType]
}

// UnmarshalText unmarshals a HasherType's enum value.
func (ht *HasherType) UnmarshalText(b []byte) error {
	s := strings.ToLower(string(b))
	OT, ok := HasherTypeUnmarshal[s]
	if !ok {
		return fmt.Errorf("%w: '%s' does not match any valid options: %s", ErrHasherTypeInvalid,
			s, ht.getKeys())
	}

	*ht = OT
	return nil
}

// getKeys returns the string keys for the HasherType enums.
func (ht HasherType) getKeys() string {
	keys := make([]string, 0, len(HasherTypeUnmarshal))
	for k := range HasherTypeUnmarshal {
		k = "'" + k + "'"
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
