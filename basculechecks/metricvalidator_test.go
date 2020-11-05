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
	"regexp"
	"testing"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/bascule"
)

func TestMetricValidatorFunc(t *testing.T) {
	goodURL, err := url.Parse("/test")
	require.Nil(t, err)
	capabilities := []string{
		"test",
		"a",
		"joweiafuoiuoiwauf",
		"it's a match",
	}
	goodAttributes := bascule.NewAttributes(map[string]interface{}{
		CapabilityKey: capabilities,
		"allowedResources": map[string]interface{}{
			"allowedPartners": []string{"meh"},
		},
	})

	tests := []struct {
		description       string
		includeAuth       bool
		attributes        bascule.Attributes
		checkCallExpected bool
		checkReason       string
		checkErr          error
		errorOut          bool
		errExpected       bool
	}{
		{
			description:       "Success",
			includeAuth:       true,
			attributes:        goodAttributes,
			checkCallExpected: true,
			errorOut:          true,
		},
		{
			description: "Include Auth Error",
			errorOut:    true,
			errExpected: true,
		},
		{
			description: "Include Auth Suppressed Error",
			errorOut:    false,
		},
		{
			description: "Prep Metrics Error",
			includeAuth: true,
			attributes:  nil,
			errorOut:    true,
			errExpected: true,
		},
		{
			description: "Prep Metrics Suppressed Error",
			includeAuth: true,
			attributes:  nil,
			errorOut:    false,
		},
		{
			description:       "Check Error",
			includeAuth:       true,
			attributes:        goodAttributes,
			checkCallExpected: true,
			checkReason:       NoCapabilitiesMatch,
			checkErr:          errors.New("test check error"),
			errorOut:          true,
			errExpected:       true,
		},
		{
			description:       "Check Suppressed Error",
			includeAuth:       true,
			attributes:        goodAttributes,
			checkCallExpected: true,
			checkReason:       NoCapabilitiesMatch,
			checkErr:          errors.New("test check error"),
			errorOut:          false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()
			auth := bascule.Authentication{
				Token: bascule.NewToken("test", "princ", tc.attributes),
				Request: bascule.Request{
					URL:    goodURL,
					Method: "GET",
				},
			}
			if tc.includeAuth {
				ctx = bascule.WithAuthentication(ctx, auth)
			}
			mockCapabilitiesChecker := new(mockCapabilitiesChecker)
			if tc.checkCallExpected {
				mockCapabilitiesChecker.On("Check", mock.Anything, mock.Anything).Return(tc.checkReason, tc.checkErr).Once()
			}

			counter := generic.NewCounter("test_capability_check")
			mockMeasures := AuthCapabilityCheckMeasures{
				CapabilityCheckOutcome: counter,
			}

			m := MetricValidator{
				C:        mockCapabilitiesChecker,
				Measures: &mockMeasures,
			}
			err := m.CreateValidator(tc.errorOut)(ctx, nil)
			mockCapabilitiesChecker.AssertExpectations(t)
			if tc.errExpected {
				assert.NotNil(err)
				return
			}
			assert.Nil(err)
		})
	}
}

func TestPrepMetrics(t *testing.T) {
	var (
		goodURL        = "/asnkfn/aefkijeoij/aiogj"
		matchingURL    = "/fnvvdsjkfji/mac:12345544322345334/geigosj"
		client         = "special"
		prepErr        = errors.New("couldn't get partner IDs from attributes")
		badValErr      = errors.New("partner IDs value not the expected string slice")
		goodEndpoint   = `/fnvvdsjkfji/.*/geigosj\b`
		goodRegex      = regexp.MustCompile(goodEndpoint)
		unusedEndpoint = `/a/b\b`
		unusedRegex    = regexp.MustCompile(unusedEndpoint)
	)

	tests := []struct {
		description       string
		noPartnerID       bool
		partnerIDs        interface{}
		url               string
		includeToken      bool
		includeAttributes bool
		includeURL        bool
		expectedPartner   string
		expectedEndpoint  string
		expectedReason    string
		expectedErr       error
	}{
		{
			description:       "Success",
			partnerIDs:        []string{"partner"},
			url:               goodURL,
			includeToken:      true,
			includeAttributes: true,
			includeURL:        true,
			expectedPartner:   "partner",
			expectedEndpoint:  "not_recognized",
			expectedReason:    "",
			expectedErr:       nil,
		},
		{
			description:       "Success Abridged URL",
			partnerIDs:        []string{"partner"},
			url:               matchingURL,
			includeToken:      true,
			includeAttributes: true,
			includeURL:        true,
			expectedPartner:   "partner",
			expectedEndpoint:  goodEndpoint,
			expectedReason:    "",
			expectedErr:       nil,
		},
		{
			description:    "Nil Token Error",
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNoToken,
		},
		{
			description:    "Nil Token Attributes Error",
			url:            goodURL,
			includeToken:   true,
			expectedReason: TokenMissingValues,
			expectedErr:    ErrNilAttributes,
		},
		{
			description:       "No Partner ID Error",
			noPartnerID:       true,
			url:               goodURL,
			includeToken:      true,
			includeAttributes: true,
			expectedPartner:   "",
			expectedEndpoint:  "",
			expectedReason:    UndeterminedPartnerID,
			expectedErr:       prepErr,
		},
		{
			description:       "Non String Slice Partner ID Error",
			partnerIDs:        []int{0, 1, 2},
			url:               goodURL,
			includeToken:      true,
			includeAttributes: true,
			expectedPartner:   "",
			expectedEndpoint:  "",
			expectedReason:    UndeterminedPartnerID,
			expectedErr:       badValErr,
		},
		{
			description:       "Non Slice Partner ID Error",
			partnerIDs:        struct{ string }{},
			url:               goodURL,
			includeToken:      true,
			includeAttributes: true,
			expectedPartner:   "",
			expectedEndpoint:  "",
			expectedReason:    UndeterminedPartnerID,
			expectedErr:       badValErr,
		},
		{
			description:       "Nil URL Error",
			partnerIDs:        []string{"partner"},
			url:               goodURL,
			includeToken:      true,
			includeAttributes: true,
			expectedPartner:   "partner",
			expectedReason:    TokenMissingValues,
			expectedErr:       ErrNoURL,
		},
	}

	m := MetricValidator{
		Endpoints: []*regexp.Regexp{unusedRegex, goodRegex},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			// setup auth
			token := bascule.NewToken("mehType", client, nil)
			if tc.includeAttributes {
				a := map[string]interface{}{
					"allowedResources": map[string]interface{}{
						"allowedPartners": tc.partnerIDs,
					},
				}

				if tc.noPartnerID {
					a["allowedResources"] = 5
				}
				attributes := bascule.NewAttributes(a)
				token = bascule.NewToken("mehType", client, attributes)
			}
			auth := bascule.Authentication{
				Authorization: "testAuth",
				Request: bascule.Request{
					Method: "get",
				},
			}
			if tc.includeToken {
				auth.Token = token
			}
			if tc.includeURL {
				u, err := url.ParseRequestURI(tc.url)
				require.Nil(err)
				auth.Request.URL = u
			}

			c, partner, endpoint, reason, err := m.prepMetrics(auth)
			if tc.includeToken {
				assert.Equal(client, c)
			}
			assert.Equal(tc.expectedPartner, partner)
			assert.Equal(tc.expectedEndpoint, endpoint)
			assert.Equal(tc.expectedReason, reason)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestDeterminePartnerMetric(t *testing.T) {
	tests := []struct {
		description    string
		partnersInput  []string
		expectedResult string
	}{
		{
			description:    "No Partners",
			expectedResult: "none",
		},
		{
			description:    "one wildcard",
			partnersInput:  []string{"*"},
			expectedResult: "wildcard",
		},
		{
			description:    "one partner",
			partnersInput:  []string{"TestPartner"},
			expectedResult: "TestPartner",
		},
		{
			description:    "many partners",
			partnersInput:  []string{"partner1", "partner2", "partner3"},
			expectedResult: "many",
		},
		{
			description:    "many partners with wildcard",
			partnersInput:  []string{"partner1", "partner2", "partner3", "*"},
			expectedResult: "wildcard",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			partner := DeterminePartnerMetric(tc.partnersInput)
			assert.Equal(tc.expectedResult, partner)
		})
	}
}
