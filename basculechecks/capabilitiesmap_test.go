/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package basculechecks

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/bascule"
)

func TestCapabilitiesMapCheck(t *testing.T) {
	goodDefault := ConstCheck("default checker")
	checkersMap := map[string]CapabilityChecker{
		"a":        ConstCheck("meh"),
		"bcedef":   ConstCheck("yay"),
		"all":      ConstCheck("good"),
		"fallback": nil,
	}
	cm := CapabilitiesMap{
		Checkers:       checkersMap,
		DefaultChecker: goodDefault,
	}
	nilCM := CapabilitiesMap{}
	goodCapabilities := []string{
		"test",
		"",
		"yay",
		"...",
	}
	goodToken := bascule.NewToken("test", "princ",
		bascule.NewAttributes(map[string]interface{}{CapabilityKey: goodCapabilities}))
	defaultCapabilities := []string{
		"test",
		"",
		"default checker",
		"...",
	}
	defaultToken := bascule.NewToken("test", "princ",
		bascule.NewAttributes(map[string]interface{}{CapabilityKey: defaultCapabilities}))
	badToken := bascule.NewToken("", "", nil)
	tests := []struct {
		description    string
		cm             CapabilitiesMap
		token          bascule.Token
		includeURL     bool
		endpoint       string
		expectedReason string
		expectedErr    error
	}{
		{
			description: "Success",
			cm:          cm,
			token:       goodToken,
			includeURL:  true,
			endpoint:    "bcedef",
		},
		{
			description: "Success Not in Map",
			cm:          cm,
			token:       defaultToken,
			includeURL:  true,
			endpoint:    "b",
		},
		{
			description: "Success Nil Map Value",
			cm:          cm,
			token:       defaultToken,
			includeURL:  true,
			endpoint:    "fallback",
		},
		{
			description:    "No Match Error",
			cm:             cm,
			token:          goodToken,
			includeURL:     true,
			endpoint:       "b",
			expectedReason: NoCapabilitiesMatch,
			expectedErr:    ErrNoValidCapabilityFound,
		},
		{
			description:    "No Match with Default Checker Error",
			cm:             cm,
			token:          defaultToken,
			includeURL:     true,
			endpoint:       "bcedef",
			expectedReason: NoCapabilitiesMatch,
			expectedErr:    ErrNoValidCapabilityFound,
		},
		{
			description:    "No Match Nil Default Checker Error",
			cm:             nilCM,
			token:          defaultToken,
			includeURL:     true,
			endpoint:       "bcedef",
			expectedReason: NoCapabilitiesMatch,
			expectedErr:    ErrNoValidCapabilityFound,
		},
		{
			description:    "No Token Error",
			cm:             cm,
			token:          nil,
			includeURL:     true,
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoToken,
		},
		{
			description:    "No Request URL Error",
			cm:             cm,
			token:          goodToken,
			includeURL:     false,
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoURL,
		},
		{
			description:    "Empty Endpoint Error",
			cm:             cm,
			token:          goodToken,
			includeURL:     true,
			endpoint:       "",
			expectedReason: EmptyParsedURL,
			expectedErr:    ErrEmptyEndpoint,
		},
		{
			description:    "Get Capabilities Error",
			cm:             cm,
			token:          badToken,
			includeURL:     true,
			endpoint:       "b",
			expectedReason: UndeterminedCapabilities,
			expectedErr:    ErrNilAttributes,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			auth := bascule.Authentication{
				Token: tc.token,
			}
			if tc.includeURL {
				goodURL, err := url.Parse("/test")
				require.Nil(err)
				auth.Request = bascule.Request{
					URL:    goodURL,
					Method: "GET",
				}
			}
			reason, err := tc.cm.Check(auth, ParsedValues{Endpoint: tc.endpoint})
			assert.Equal(tc.expectedReason, reason)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
				return
			}
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}
