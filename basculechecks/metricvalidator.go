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
	"fmt"
	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/bascule"
)

var defaultLogger = log.NewNopLogger()

// CapabilitiesChecker is an object that can determine if a request is
// authorized given a bascule.Authentication object.  If it's not authorized,
// a reason and error are given for logging and metrics.
type CapabilitiesChecker interface {
	Check(auth bascule.Authentication, vals ParsedValues) (string, error)
}

type ParsedValues struct {
	Endpoint string
	Partner  string
}

// MetricValidator determines if a request is authorized and then updates a
// metric to show those results.
type MetricValidator struct {
	C         CapabilitiesChecker
	Measures  *AuthCapabilityCheckMeasures
	Endpoints []*regexp.Regexp
}

// CreateValidator provides a function for authorization middleware.  The
// function parses the information needed for the CapabilitiesChecker, calls it
// to determine if the request is authorized, and maintains the results in a
// metric.  The function can actually mark the request as unauthorized or just
// update the metric and allow the request, depending on configuration.  This
// allows for monitoring before being more strict with authorization.
func (m MetricValidator) CreateValidator(errorOut bool) bascule.ValidatorFunc {
	return func(ctx context.Context, _ bascule.Token) error {
		// if we're not supposed to error out, the outcome should be accepted on failure
		failureOutcome := AcceptedOutcome
		if errorOut {
			// if we actually error out, the outcome is the request being rejected
			failureOutcome = RejectedOutcome
		}

		auth, ok := bascule.FromContext(ctx)
		if !ok {
			m.Measures.CapabilityCheckOutcome.With(OutcomeLabel, failureOutcome, ReasonLabel, TokenMissing, ClientIDLabel, "", PartnerIDLabel, "", EndpointLabel, "").Add(1)
			if errorOut {
				return ErrNoAuth
			}
			return nil
		}

		client, partnerID, endpoint, reason, err := m.prepMetrics(auth)
		labels := []string{ClientIDLabel, client, PartnerIDLabel, partnerID, EndpointLabel, endpoint}
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			m.Measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		v := ParsedValues{
			Endpoint: endpoint,
			Partner:  partnerID,
		}

		reason, err = m.C.Check(auth, v)
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			m.Measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		labels = append(labels, OutcomeLabel, AcceptedOutcome, ReasonLabel, "")
		m.Measures.CapabilityCheckOutcome.With(labels...).Add(1)
		return nil
	}
}

// prepMetrics gathers the information needed for metric label information.  It
// gathers the client ID, partnerID, and endpoint (bucketed) for more information
// on the metric when a request is unauthorized.
func (m MetricValidator) prepMetrics(auth bascule.Authentication) (string, string, string, string, error) {
	if auth.Token == nil {
		return "", "", "", TokenMissingValues, ErrNoToken
	}
	client := auth.Token.Principal()
	if auth.Token.Attributes() == nil {
		return client, "", "", TokenMissingValues, ErrNilAttributes
	}
	partnerVal, ok := bascule.GetNestedAttribute(auth.Token.Attributes(), PartnerKeys()...)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("couldn't get partner IDs from attributes using keys %v", PartnerKeys())
	}
	partnerIDs, ok := partnerVal.([]string)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("partner IDs value not the expected string slice: %v", partnerVal)
	}
	partnerID := DeterminePartnerMetric(partnerIDs)
	if auth.Request.URL == nil {
		return client, partnerID, "", TokenMissingValues, ErrNoURL
	}
	escapedURL := auth.Request.URL.EscapedPath()
	endpoint := determineEndpointMetric(m.Endpoints, escapedURL)
	return client, partnerID, endpoint, "", nil
}

// DeterminePartnerMetric takes a list of partners and decides what the partner
// metric label should be.
func DeterminePartnerMetric(partners []string) string {
	if len(partners) < 1 {
		return "none"
	}
	if len(partners) == 1 {
		if partners[0] == "*" {
			return "wildcard"
		}
		return partners[0]
	}
	for _, partner := range partners {
		if partner == "*" {
			return "wildcard"
		}
	}
	return "many"
}

// determineEndpointMetric takes a list of regular expressions and applies them
// to the url of the request to decide what the endpoint metric label should be.
func determineEndpointMetric(endpoints []*regexp.Regexp, urlHit string) string {
	for _, r := range endpoints {
		idxs := r.FindStringIndex(urlHit)
		if idxs == nil {
			continue
		}
		if idxs[0] == 0 {
			return r.String()
		}
	}
	return "not_recognized"
}
