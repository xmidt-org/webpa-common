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
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/bascule"
)

func TestCapabilitiesChecker(t *testing.T) {
	var v interface{}
	v = CapabilitiesValidator{}
	_, ok := v.(CapabilitiesChecker)
	assert.True(t, ok)
}

func TestCapabilitiesValidatorFunc(t *testing.T) {
	capabilities := []string{
		"test",
		"a",
		"joweiafuoiuoiwauf",
		"it's a match",
	}
	goodURL, err := url.Parse("/test")
	require.Nil(t, err)
	goodRequest := bascule.Request{
		URL:    goodURL,
		Method: "GET",
	}
	tests := []struct {
		description  string
		includeAuth  bool
		includeToken bool
		errorOut     bool
		errExpected  bool
	}{
		{
			description:  "Success",
			includeAuth:  true,
			includeToken: true,
			errorOut:     true,
		},
		{
			description: "No Auth Error",
			errorOut:    true,
			errExpected: true,
		},
		{
			description: "No Auth Suppressed Error",
		},
		{
			description: "Check Error",
			includeAuth: true,
			errorOut:    true,
			errExpected: true,
		},
		{
			description: "Check Suppressed Error",
			includeAuth: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			ctx := context.Background()
			auth := bascule.Authentication{
				Request: goodRequest,
			}
			if tc.includeToken {
				auth.Token = bascule.NewToken("test", "princ",
					bascule.NewAttributes(map[string]interface{}{CapabilityKey: capabilities}))
			}
			if tc.includeAuth {
				ctx = bascule.WithAuthentication(ctx, auth)
			}
			c := CapabilitiesValidator{
				Checker: ConstCheck("it's a match"),
			}
			err := c.CreateValidator(tc.errorOut)(ctx, bascule.NewToken("", "", nil))
			if tc.errExpected {
				assert.NotNil(err)
				return
			}
			assert.Nil(err)
		})
	}
}

func TestCapabilitiesValidatorCheck(t *testing.T) {
	capabilities := []string{
		"test",
		"a",
		"joweiafuoiuoiwauf",
		"it's a match",
	}
	pv := ParsedValues{}
	tests := []struct {
		description       string
		includeToken      bool
		includeAttributes bool
		includeURL        bool
		checker           CapabilityChecker
		expectedReason    string
		expectedErr       error
	}{
		{
			description:       "Success",
			includeAttributes: true,
			includeURL:        true,
			checker:           ConstCheck("it's a match"),
			expectedErr:       nil,
		},
		{
			description:    "No Token Error",
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoToken,
		},
		{
			description:    "Get Capabilities Error",
			includeToken:   true,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    ErrNilAttributes,
		},
		{
			description:       "No URL Error",
			includeAttributes: true,
			expectedReason:    TokenMissingValues,
			expectedErr:       ErrNoURL,
		},
		{
			description:       "Check Capabilities Error",
			includeAttributes: true,
			includeURL:        true,
			checker:           AlwaysCheck(false),
			expectedReason:    NoCapabilitiesMatch,
			expectedErr:       ErrNoValidCapabilityFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			c := CapabilitiesValidator{
				Checker: tc.checker,
			}
			a := bascule.Authentication{}
			if tc.includeToken {
				a.Token = bascule.NewToken("", "", nil)
			}
			if tc.includeAttributes {
				a.Token = bascule.NewToken("test", "princ",
					bascule.NewAttributes(map[string]interface{}{CapabilityKey: capabilities}))
			}
			if tc.includeURL {
				goodURL, err := url.Parse("/test")
				require.Nil(err)
				a.Request = bascule.Request{
					URL:    goodURL,
					Method: "GET",
				}
			}
			reason, err := c.Check(a, pv)
			assert.Equal(tc.expectedReason, reason)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
				return
			}
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}

func TestCheckCapabilities(t *testing.T) {
	capabilities := []string{
		"test",
		"a",
		"joweiafuoiuoiwauf",
		"it's a match",
	}

	tests := []struct {
		description    string
		goodCapability string
		expectedErr    error
	}{
		{
			description:    "Success",
			goodCapability: "it's a match",
		},
		{
			description: "No Capability Found Error",
			expectedErr: ErrNoValidCapabilityFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			c := CapabilitiesValidator{
				Checker: ConstCheck(tc.goodCapability),
			}
			err := c.checkCapabilities(capabilities, "", "")
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
				return
			}
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	goodKeyVal := []string{"cap1", "cap2"}
	emptyVal := []string{}
	getCapabilitiesErr := errors.New("couldn't get capabilities using key")
	badCapabilitiesErr := errors.New("capabilities value not the expected string slice")
	tests := []struct {
		description      string
		nilAttributes    bool
		missingAttribute bool
		keyValue         interface{}
		expectedVals     []string
		expectedReason   string
		expectedErr      error
	}{
		{
			description:    "Success",
			keyValue:       goodKeyVal,
			expectedVals:   goodKeyVal,
			expectedReason: "",
			expectedErr:    nil,
		},
		{
			description:    "Nil Attributes Error",
			nilAttributes:  true,
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    ErrNilAttributes,
		},
		{
			description:      "No Attribute Error",
			missingAttribute: true,
			expectedVals:     emptyVal,
			expectedReason:   UndeterminedCapabilities,
			expectedErr:      getCapabilitiesErr,
		},
		{
			description:    "Nil Capabilities Error",
			keyValue:       nil,
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    badCapabilitiesErr,
		},
		{
			description:    "Non List Capabilities Error",
			keyValue:       struct{ string }{"abcd"},
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    badCapabilitiesErr,
		},
		{
			description:    "Non String List Capabilities Error",
			keyValue:       []int{0, 1, 2},
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    badCapabilitiesErr,
		},
		{
			description:    "Empty Capabilities Error",
			keyValue:       emptyVal,
			expectedVals:   emptyVal,
			expectedReason: EmptyCapabilitiesList,
			expectedErr:    ErrNoVals,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			m := map[string]interface{}{CapabilityKey: tc.keyValue}
			if tc.missingAttribute {
				m = map[string]interface{}{}
			}
			attributes := bascule.NewAttributes(m)
			if tc.nilAttributes {
				attributes = nil
			}
			vals, reason, err := getCapabilities(attributes)
			assert.Equal(tc.expectedVals, vals)
			assert.Equal(tc.expectedReason, reason)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}
