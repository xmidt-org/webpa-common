// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiAccessor(t *testing.T) {
	var (
		uniquenessCheck = HasherTypeMarshal
		opts            options
	)

	for t := range maps.Keys(uniquenessCheck) {
		opts = append(opts, Hasher(t))
	}

	tests := []struct {
		url             string
		expectedNormMap map[HasherType]map[string]string
	}{
		{
			url: "https://1.2.3.4",
			expectedNormMap: map[HasherType]map[string]string{
				RawURLType:   {"https://1.2.3.4": "https://1.2.3.4"},
				HostnameType: {"1.2.3.4": "https://1.2.3.4"},
			},
		},
		{
			url: "https://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000",
			expectedNormMap: map[HasherType]map[string]string{
				RawURLType:   {"https://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000": "https://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000"},
				HostnameType: {"2001:0db8:85a3:0000:0000:8a2e:0370:7334": "https://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000"},
			},
		},
		{
			url: "http://example.com:80",
			expectedNormMap: map[HasherType]map[string]string{
				RawURLType:   {"http://example.com:80": "http://example.com:80"},
				HostnameType: {"example.com": "http://example.com:80"},
			},
		},
		{
			url: "https://some.super.long.domain.example.com:8080/somepath",
			expectedNormMap: map[HasherType]map[string]string{
				RawURLType:   {"https://some.super.long.domain.example.com:8080/somepath": "https://some.super.long.domain.example.com:8080/somepath"},
				HostnameType: {"some.super.long.domain.example.com": "https://some.super.long.domain.example.com:8080/somepath"},
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("MultiAccessor (all hashers enabled): [%d]", i), func(t *testing.T) {
			require := require.New(t)
			hs := New(opts).(multiHasher)
			require.NotEmpty(hs)
			hs.Add(tc.url)
			for ht, expected := range tc.expectedNormMap {
				t.Run(ht.String(), func(t *testing.T) {
					assert := assert.New(t)
					nm := hs.GetNormMap()[ht]
					assert.NotEmpty(nm)
					assert.Equal(expected, nm)
				})
			}
		})
	}

}
