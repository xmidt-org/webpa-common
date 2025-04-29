// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostnameNormalizerPanics(t *testing.T) {

	tests := []struct {
		description string
		url         string
	}{
		{
			"formating error",
			"[[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000]",
		},
		{
			"port formating error",
			"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:::::17000",
		},
		{
			"missing host error",
			"https://",
		},
		{
			"missing schema error",
			"example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Panics(func() {
				HostnameNormalizer(tc.url)
			})
		})
	}
}
