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

func TestNewCapabilitiesMap(t *testing.T) {
	goodCheckers := map[string]CapabilityChecker{
		"a":      ConstCheck("meh"),
		"bcedef": ConstCheck("yay"),
		"all":    ConstCheck("good"),
	}
	emptyCheckers := map[string]CapabilityChecker{}
	goodDefault := ConstCheck("default checker")
	tests := []struct {
		description    string
		goodDefault    bool
		checkersMap    map[string]CapabilityChecker
		expectedStruct *CapabilitiesMap
		expectedErr    error
	}{
		{
			description: "Success",
			goodDefault: true,
			checkersMap: goodCheckers,
			expectedStruct: &CapabilitiesMap{
				checkers:       goodCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description: "Success with Empty Checkers",
			goodDefault: true,
			checkersMap: emptyCheckers,
			expectedStruct: &CapabilitiesMap{
				checkers:       emptyCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description: "Success with Nil Checkers",
			goodDefault: true,
			checkersMap: nil,
			expectedStruct: &CapabilitiesMap{
				checkers:       emptyCheckers,
				defaultChecker: goodDefault,
			},
		},
		{
			description:    "Nil Default Error",
			checkersMap:    goodCheckers,
			expectedStruct: nil,
			expectedErr:    ErrNilDefaultChecker,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			var d CapabilityChecker
			if tc.goodDefault {
				d = goodDefault
			}
			c, err := NewCapabilitiesMap(tc.checkersMap, d)
			assert.Equal(tc.expectedStruct, c)
			assert.Equal(tc.expectedErr, err)
		})
	}
}

func TestCapabilitiesMapCheck(t *testing.T) {
	goodDefault := ConstCheck("default checker")
	checkersMap := map[string]CapabilityChecker{
		"a":      ConstCheck("meh"),
		"bcedef": ConstCheck("yay"),
		"all":    ConstCheck("good"),
	}
	cm, err := NewCapabilitiesMap(checkersMap, goodDefault)
	require.Nil(t, err)
	require.NotNil(t, cm)
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
		token          bascule.Token
		includeURL     bool
		endpoint       string
		expectedReason string
		expectedErr    error
	}{
		{
			description: "Success",
			token:       goodToken,
			includeURL:  true,
			endpoint:    "bcedef",
		},
		{
			description: "Success with Default Checker",
			token:       defaultToken,
			includeURL:  true,
			endpoint:    "b",
		},
		{
			description:    "No Match Error",
			token:          goodToken,
			includeURL:     true,
			endpoint:       "b",
			expectedReason: NoCapabilitiesMatch,
			expectedErr:    ErrNoValidCapabilityFound,
		},
		{
			description:    "No Match with Default Checker Error",
			token:          defaultToken,
			includeURL:     true,
			endpoint:       "bcedef",
			expectedReason: NoCapabilitiesMatch,
			expectedErr:    ErrNoValidCapabilityFound,
		},
		{
			description:    "No Token Error",
			token:          nil,
			includeURL:     true,
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoToken,
		},
		{
			description:    "No Request URL Error",
			token:          goodToken,
			includeURL:     false,
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoURL,
		},
		{
			description:    "Empty Endpoint Error",
			token:          goodToken,
			includeURL:     true,
			endpoint:       "",
			expectedReason: EmptyParsedURL,
			expectedErr:    ErrEmptyEndpoint,
		},
		{
			description:    "Get Capabilities Error",
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
			reason, err := cm.Check(auth, ParsedValues{Endpoint: tc.endpoint})
			assert.Equal(tc.expectedReason, reason)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
				return
			}
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}
