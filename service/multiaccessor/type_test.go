// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeUnmarshalling(t *testing.T) {
	tests := []struct {
		description string
		config      []byte
		invalid     bool
	}{
		{
			description: "UnknownType valid",
			config:      []byte("unknown"),
		},
		{
			description: "RawURLType valid",
			config:      []byte("raw_url"),
		},
		{
			description: "HostnameType valid",
			config:      []byte("hostname"),
		},
		{
			description: "Nonexistent type invalid",
			config:      []byte("FOOBAR"),
			invalid:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			var l HasherType

			err := l.UnmarshalText(tc.config)
			assert.NotEmpty(l.getKeys())
			if tc.invalid {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(string(tc.config), l.String())
			}
		})
	}
}

func TestTypeState(t *testing.T) {
	tests := []struct {
		description string
		val         HasherType
		expectedVal string
		invalid     bool
		empty       bool
	}{
		{
			description: "UnknownType valid",
			val:         UnknownType,
			expectedVal: "unknown",
			empty:       true,
			invalid:     true,
		},
		{
			description: "RawURLType valid",
			val:         RawURLType,
			expectedVal: "raw_url",
		},
		{
			description: "HostnameType valid",
			val:         HostnameType,
			expectedVal: "hostname",
		},
		{
			description: "lastLevel valid",
			val:         lastType,
			expectedVal: "unknown",
			invalid:     true,
		},
		{
			description: "Nonexistent positive HasherType invalid",
			val:         lastType + 1,
			expectedVal: "unknown",
			invalid:     true,
		},
		{
			description: "Nonexistent negative HasherType invalid",
			val:         UnknownType - 1,
			expectedVal: "unknown",
			invalid:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.expectedVal, tc.val.String())
			assert.Equal(!tc.invalid, tc.val.IsValid())
			assert.Equal(tc.empty, tc.val.IsEmpty())
		})
	}
}
