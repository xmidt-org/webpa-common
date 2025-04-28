// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package multiaccessor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		description  string
		expectedUrls []string
		jsonConfig   []byte
	}{
		{
			description:  "empty configuration success",
			expectedUrls: []string{"https://example.com"},
			jsonConfig:   []byte("{}"),
		},
		{
			description: "single hasher configuration success",
			jsonConfig: []byte(`{
				"hashers": [
					{
						"type": "raw_url"
					}
				]
			}`),
			expectedUrls: []string{"https://example.com"},
		},
		{
			description: "single disabled hasher configuration success (defaults to host_name hasher)",
			jsonConfig: []byte(`{
				"hashers": [
					{
						"type": "raw_url",
						"disable": true
					}
				]
			}`),
			expectedUrls: []string{"https://example.com"},
		},
		{
			description: "multi hasher configuration success",
			jsonConfig: []byte(`{
				"hashers": [
					{
						"type": "raw_url"
					},
					{
						"type": "hostname"
					}
				]
			}`),
			expectedUrls: []string{"https://example.com", "https://example.com"},
		},
		{
			description: "multi hasher (1 disabled hasher) configuration success",
			jsonConfig: []byte(`{
				"hashers": [
					{
						"type": "raw_url",
						"disable": true
					},
					{
						"type": "hostname"
					}
				]
			}`),
			expectedUrls: []string{"https://example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			b := Builder{}
			require.NoError(json.Unmarshal(tc.jsonConfig, &b))
			hs := b.Build(VnodeCount(111), Instances([]string{"https://example.com"}))
			require.NotNil(hs)
			url, err := hs.Get([]byte("test"))
			assert.NotEmpty(url)
			assert.NoError(err)
			for _, k := range []string{"a", "alsdkjfa;lksehjuro8iwurjhf", "asdf8974", "875kjh4", "928375hjdfgkyu9832745kjshdfgoi873465"} {
				i, err := hs.Get([]byte(k))
				assert.Equal(tc.expectedUrls, i)
				assert.NoError(err)
			}

			require.NotNil(hs)
		})
	}
}
