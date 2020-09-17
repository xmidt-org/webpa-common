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

type CapabilitiesChecker interface {
	Check(auth bascule.Authentication) (string, error)
}

type MetricValidator struct {
	C         CapabilitiesChecker
	Measures  *AuthCapabilityCheckMeasures
	Endpoints []*regexp.Regexp
}

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

		reason, err = m.C.Check(auth)
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

// prepMetrics gathers the information needed for metric label information.
func (m MetricValidator) prepMetrics(auth bascule.Authentication) (string, string, string, string, error) {
	if auth.Token == nil {
		return "", "", "", TokenMissingValues, ErrNoToken
	}
	client := auth.Token.Principal()
	if auth.Token.Attributes() == nil {
		return client, "", "", TokenMissingValues, ErrNilAttributes
	}
	partnerIDs, ok := auth.Token.Attributes().GetStringSlice(PartnerKey)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("couldn't get partner IDs from attributes using key %v", PartnerKey)
	}
	partnerID := determinePartnerMetric(partnerIDs)
	if auth.Request.URL == nil {
		return client, partnerID, "", TokenMissingValues, ErrNoURL
	}
	escapedURL := auth.Request.URL.EscapedPath()
	endpoint := determineEndpointMetric(m.Endpoints, escapedURL)
	return client, partnerID, endpoint, "", nil
}

// determinePartnerMetric takes a list of partners and decides what the partner
// metric label should be.
func determinePartnerMetric(partners []string) string {
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
