package basculechecks

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
)

func TestCreateBasculeCheck(t *testing.T) {
	urlStr := "/good/url/for/testing"
	goodURL, err := url.Parse(urlStr)
	assert.Nil(t, err)
	goodPrincipal := "party:ppl"
	goodCapabilities := []string{
		"test:/good/for/nothing:all",
		`test:/good/.*/testing\b:all`,
	}
	goodPartner := []string{"meh"}

	tests := []struct {
		description      string
		errorOut         bool
		emptyContext     bool
		partnerIDs       interface{}
		capabilities     interface{}
		expectedOutcome  string
		expectedReason   string
		expectedClient   string
		expectedPartner  string
		expectedEndpoint string
		expectedErr      error
	}{
		{
			description:      "Success",
			partnerIDs:       goodPartner,
			capabilities:     goodCapabilities,
			expectedOutcome:  AcceptedOutcome,
			expectedReason:   "",
			expectedClient:   goodPrincipal,
			expectedPartner:  goodPartner[0],
			expectedEndpoint: urlStr,
		},
		{
			description:     "Token Missing From Context Error",
			errorOut:        true,
			emptyContext:    true,
			partnerIDs:      goodPartner,
			capabilities:    goodCapabilities,
			expectedOutcome: RejectedOutcome,
			expectedReason:  TokenMissing,
			expectedErr:     ErrNoAuth,
		},
		{
			description:     "Token Missing From Context Accepted",
			emptyContext:    true,
			partnerIDs:      goodPartner,
			capabilities:    goodCapabilities,
			expectedOutcome: AcceptedOutcome,
			expectedReason:  TokenMissing,
		},
		{
			description:     "Prep Metrics Error",
			errorOut:        true,
			partnerIDs:      []int{5, 7, 11},
			capabilities:    goodCapabilities,
			expectedOutcome: RejectedOutcome,
			expectedReason:  UndeterminedPartnerID,
			expectedClient:  goodPrincipal,
			expectedErr:     errors.New("couldn't get partner IDs"),
		},
		{
			description:     "Prep Metrics Accepted",
			partnerIDs:      []int{5, 7, 11},
			capabilities:    goodCapabilities,
			expectedOutcome: AcceptedOutcome,
			expectedReason:  UndeterminedPartnerID,
			expectedClient:  goodPrincipal,
		},
		{
			description:      "Get Capabilities Error",
			errorOut:         true,
			partnerIDs:       goodPartner,
			capabilities:     []int{3, 1, 4},
			expectedOutcome:  RejectedOutcome,
			expectedReason:   UndeterminedCapabilities,
			expectedClient:   goodPrincipal,
			expectedPartner:  goodPartner[0],
			expectedEndpoint: urlStr,
			expectedErr:      errors.New("couldn't get capabilities"),
		},
		{
			description:      "Get Capabilities Accepted",
			partnerIDs:       goodPartner,
			capabilities:     []int{3, 1, 4},
			expectedOutcome:  AcceptedOutcome,
			expectedReason:   UndeterminedCapabilities,
			expectedClient:   goodPrincipal,
			expectedPartner:  goodPartner[0],
			expectedEndpoint: urlStr,
		},
		{
			description:      "Capability Check Error",
			errorOut:         true,
			partnerIDs:       goodPartner,
			capabilities:     []string{"failure"},
			expectedOutcome:  RejectedOutcome,
			expectedReason:   NoCapabilitiesMatch,
			expectedClient:   goodPrincipal,
			expectedPartner:  goodPartner[0],
			expectedEndpoint: urlStr,
			expectedErr:      ErrNoValidCapabilityFound,
		},
		{
			description:      "Capability Check Accepted",
			partnerIDs:       goodPartner,
			capabilities:     []string{"failure"},
			expectedOutcome:  AcceptedOutcome,
			expectedReason:   NoCapabilitiesMatch,
			expectedClient:   goodPrincipal,
			expectedPartner:  goodPartner[0],
			expectedEndpoint: urlStr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			p := xmetricstest.NewProvider(nil, Metrics)
			m := NewAuthCapabilityCheckMeasures(p)

			c, err := NewCapabilityChecker(m, "test:", "all")
			assert.Nil(err)

			attrMap := map[string]interface{}{PartnerKey: tc.partnerIDs, CapabilityKey: tc.capabilities}
			attrs := bascule.NewAttributesFromMap(attrMap)
			auth := bascule.Authentication{
				Authorization: "TestAuthorization",
				Token:         bascule.NewToken("cool type", goodPrincipal, attrs),
				Request: bascule.Request{
					URL:    goodURL,
					Method: "GET",
				},
			}
			ctx := context.Background()
			if !tc.emptyContext {
				ctx = bascule.WithAuthentication(ctx, auth)
			}

			p.Assert(t, AuthCapabilityCheckOutcome)(xmetricstest.Value(0.0))
			check := c.CreateBasculeCheck(tc.errorOut)
			err = check(ctx, auth.Token)
			p.Assert(t, AuthCapabilityCheckOutcome,
				OutcomeLabel, tc.expectedOutcome,
				ReasonLabel, tc.expectedReason,
				ClientIDLabel, tc.expectedClient,
				PartnerIDLabel, tc.expectedPartner,
				EndpointLabel, tc.expectedEndpoint,
			)(xmetricstest.Counter, xmetricstest.Value(1.0))
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestNewCapabilityChecker(t *testing.T) {
	tests := []struct {
		description  string
		goodMeasures bool
		prefix       string
		expectedErr  error
	}{
		{
			description:  "Success",
			goodMeasures: true,
			prefix:       "testprefix:",
			expectedErr:  nil,
		},
		{
			description:  "Nil Measures Error",
			goodMeasures: false,
			prefix:       "",
			expectedErr:  errors.New("nil capability check measures"),
		},
		{
			description:  "Bad Prefix Error",
			goodMeasures: true,
			prefix:       `\K`,
			expectedErr:  errors.New("failed to compile prefix given"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			var m *AuthCapabilityCheckMeasures
			if tc.goodMeasures {
				p := xmetricstest.NewProvider(nil, Metrics)
				m = NewAuthCapabilityCheckMeasures(p)
			}
			check, err := NewCapabilityChecker(m, tc.prefix, "all")
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
				assert.NotNil(check)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
				assert.Nil(check)
			}
		})
	}
}

func TestCheckCapabilities(t *testing.T) {
	goodURL, err := url.Parse("/test")
	assert.Nil(t, err)
	goodRequest := bascule.Request{
		URL:    goodURL,
		Method: "GET",
	}
	goodCapabilities := []string{
		"d:e:f:/aaaa:post",
		"a:b:d:/aaaa:allIn",
		`a:b:c:/test\b:post`,
		`a:b:c:z:allIn`,
		`a:b:c:/test\b:get`,
	}
	badCapabilities := []string{
		"a:b:d:/aaaa:allIn",
		`a:b:c:/test\b:post`,
	}
	tests := []struct {
		description  string
		capabilities []string
		expectedErr  error
	}{
		{
			description:  "Success",
			capabilities: goodCapabilities,
			expectedErr:  nil,
		},
		{
			description:  "No Capabilities Error",
			capabilities: []string{},
			expectedErr:  ErrNoValidCapabilityFound,
		},
		{
			description:  "No Matching Capabilities Error",
			capabilities: badCapabilities,
			expectedErr:  ErrNoValidCapabilityFound,
		},
	}
	p := xmetricstest.NewProvider(nil, Metrics)
	m := NewAuthCapabilityCheckMeasures(p)
	c, err := NewCapabilityChecker(m, "a:b:c:", "allIn")
	assert.Nil(t, err)

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := c.checkCapabilities(tc.capabilities, goodRequest)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestPrepMetrics(t *testing.T) {
	goodURL := "/asnkfn/aefkijeoij/aiogj"
	matchingURL := "/fnvvdsjkfji/mac:12345544322345334/geigosj"
	abridgedURL := "/fnvvdsjkfji/.../geigosj"
	client := "special"
	prepErr := errors.New("couldn't get partner IDs from attributes")
	tests := []struct {
		description      string
		noPartnerID      bool
		partnerIDs       interface{}
		url              string
		expectedPartner  string
		expectedEndpoint string
		expectedReason   string
		expectedErr      error
	}{
		{
			description:      "Success",
			partnerIDs:       []string{"partner"},
			url:              goodURL,
			expectedPartner:  "partner",
			expectedEndpoint: goodURL,
			expectedReason:   "",
			expectedErr:      nil,
		},
		{
			description:      "Success Abridged URL",
			partnerIDs:       []string{"partner"},
			url:              matchingURL,
			expectedPartner:  "partner",
			expectedEndpoint: abridgedURL,
			expectedReason:   "",
			expectedErr:      nil,
		},
		{
			description:      "No Partner ID Error",
			noPartnerID:      true,
			url:              goodURL,
			expectedPartner:  "",
			expectedEndpoint: "",
			expectedReason:   UndeterminedPartnerID,
			expectedErr:      prepErr,
		},
		{
			description:      "Non String Slice Partner ID Error",
			partnerIDs:       []int{0, 1, 2},
			url:              goodURL,
			expectedPartner:  "",
			expectedEndpoint: "",
			expectedReason:   UndeterminedPartnerID,
			expectedErr:      prepErr,
		},
		{
			description:      "Non Slice Partner ID Error",
			partnerIDs:       struct{ string }{},
			url:              goodURL,
			expectedPartner:  "",
			expectedEndpoint: "",
			expectedReason:   UndeterminedPartnerID,
			expectedErr:      prepErr,
		},
	}

	p := xmetricstest.NewProvider(nil, Metrics)
	m := NewAuthCapabilityCheckMeasures(p)
	c, err := NewCapabilityChecker(m, "prefix:", "all")
	assert.Nil(t, err)

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.Nil(err)
			m := map[string]interface{}{}
			if !tc.noPartnerID {
				m[PartnerKey] = tc.partnerIDs
			}
			attributes := bascule.NewAttributesFromMap(m)
			auth := bascule.Authentication{
				Authorization: "testAuth",
				Token:         bascule.NewToken("mehType", client, attributes),
				Request: bascule.Request{
					URL:    u,
					Method: "get",
				},
			}
			c, partner, endpoint, reason, err := c.prepMetrics(auth)
			assert.Equal(client, c)
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
			partner := determinePartnerMetric(tc.partnersInput)
			assert.Equal(tc.expectedResult, partner)
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	goodKeyVal := []string{"cap1", "cap2"}
	emptyVal := []string{}
	getCapabilitiesErr := errors.New("couldn't get capabilities using key")
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
			expectedErr:    getCapabilitiesErr,
		},
		{
			description:    "Non List Capabilities Error",
			keyValue:       struct{ string }{"abcd"},
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    getCapabilitiesErr,
		},
		{
			description:    "Non String List Capabilities Error",
			keyValue:       []int{0, 1, 2},
			expectedVals:   emptyVal,
			expectedReason: UndeterminedCapabilities,
			expectedErr:    getCapabilitiesErr,
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
			attributes := bascule.NewAttributesFromMap(m)
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
