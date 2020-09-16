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
	"fmt"
	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/logging"
)

var (
	ErrNilChecker  = errors.New("capability checker cannot be nil")
	ErrNilMeasures = errors.New("capability check measures cannot be nil")
)

var defaultLogger = log.NewNopLogger()

type CapabilityChecker interface {
	Check(auth bascule.Authentication) (string, error)
}

type metricChecker struct {
	c         CapabilityChecker
	measures  *AuthCapabilityCheckMeasures
	endpoints []*regexp.Regexp
}

func (m *metricChecker) CreateBasculeCheck(errorOut bool) bascule.ValidatorFunc {
	return func(ctx context.Context, _ bascule.Token) error {
		// if we're not supposed to error out, the outcome should be accepted on failure
		failureOutcome := AcceptedOutcome
		if errorOut {
			// if we actually error out, the outcome is the request being rejected
			failureOutcome = RejectedOutcome
		}

		auth, ok := bascule.FromContext(ctx)
		if !ok {
			m.measures.CapabilityCheckOutcome.With(OutcomeLabel, failureOutcome, ReasonLabel, TokenMissing, ClientIDLabel, "", PartnerIDLabel, "", EndpointLabel, "").Add(1)
			if errorOut {
				return ErrNoAuth
			}
			return nil
		}

		client, partnerID, endpoint, reason, err := m.prepMetrics(auth)
		labels := []string{ClientIDLabel, client, PartnerIDLabel, partnerID, EndpointLabel, endpoint}
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			m.measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		reason, err = m.c.Check(auth)
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			m.measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		labels = append(labels, OutcomeLabel, AcceptedOutcome, ReasonLabel, "")
		m.measures.CapabilityCheckOutcome.With(labels...).Add(1)
		return nil
	}
}

func NewMetricCapabilityChecker(c CapabilityChecker, m *AuthCapabilityCheckMeasures, endpoints []*regexp.Regexp) (*metricChecker, error) {
	if c == nil {
		return nil, ErrNilChecker
	}
	if m == nil {
		return nil, ErrNilMeasures
	}
	mc := metricChecker{
		measures:  m,
		c:         c,
		endpoints: endpoints,
	}
	return &mc, nil
}

func NewMetricCapabilityCheckerFromStrings(c CapabilityChecker, m *AuthCapabilityCheckMeasures, endpoints []string, logger log.Logger) (*metricChecker, error) {
	// there's no point in compiling these regular expressions if things are nil.
	if c == nil {
		return nil, ErrNilChecker
	}
	if m == nil {
		return nil, ErrNilMeasures
	}

	var endpointRegexps []*regexp.Regexp
	l := logger
	if logger == nil {
		l = defaultLogger
	}
	for _, e := range endpoints {
		r, err := regexp.Compile(e)
		if err != nil {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "failed to compile regular expression", "regex", e, logging.ErrorKey(), err.Error())
			continue
		}
		endpointRegexps = append(endpointRegexps, r)
	}

	return NewMetricCapabilityChecker(c, m, endpointRegexps)

}

// prepMetrics gathers the information needed for metric label information.
func (m *metricChecker) prepMetrics(auth bascule.Authentication) (string, string, string, string, error) {
	// getting metric information
	client := auth.Token.Principal()
	partnerIDs, ok := auth.Token.Attributes().GetStringSlice(PartnerKey)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("couldn't get partner IDs from attributes using key %v", PartnerKey)
	}
	partnerID := determinePartnerMetric(partnerIDs)
	escapedURL := auth.Request.URL.EscapedPath()
	endpoint := determineEndpointMetric(m.endpoints, escapedURL)
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
