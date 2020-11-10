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
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/prometheus/client_golang/prometheus"
	themisXmetrics "github.com/xmidt-org/themis/xmetrics"
	"github.com/xmidt-org/webpa-common/xmetrics"

	"go.uber.org/fx"
)

// Names for our metrics
const (
	AuthCapabilityCheckOutcome = "auth_capability_check"
)

// labels
const (
	OutcomeLabel   = "outcome"
	ReasonLabel    = "reason"
	ClientIDLabel  = "clientid"
	EndpointLabel  = "endpoint"
	PartnerIDLabel = "partnerid"
)

// outcomes
const (
	RejectedOutcome = "rejected"
	AcceptedOutcome = "accepted"
	// reasons
	TokenMissing             = "auth_missing"
	UndeterminedPartnerID    = "undetermined_partner_ID"
	UndeterminedCapabilities = "undetermined_capabilities"
	EmptyCapabilitiesList    = "empty_capabilities_list"
	TokenMissingValues       = "auth_is_missing_values"
	NoCapabilityChecker      = "no_capability_checker"
	NoCapabilitiesMatch      = "no_capabilities_match"
	EmptyParsedURL           = "empty_parsed_URL"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       AuthCapabilityCheckOutcome,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the capability checker, providing outcome information by client, partner, and endpoint",
			LabelNames: []string{OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel},
		},
	}
}

func ProvideMetrics() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounter(prometheus.CounterOpts{
			Name:        AuthCapabilityCheckOutcome,
			Help:        "Counter for the capability checker, providing outcome information by client, partner, and endpoint",
			ConstLabels: nil,
		}, OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel),
	)
}

// AuthCapabilityCheckMeasures describes the defined metrics that will be used by clients
type AuthCapabilityCheckMeasures struct {
	fx.In

	CapabilityCheckOutcome metrics.Counter `name:"auth_capability_check"`
}

// NewAuthCapabilityCheckMeasures realizes desired metrics
func NewAuthCapabilityCheckMeasures(p provider.Provider) *AuthCapabilityCheckMeasures {
	return &AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: p.NewCounter(AuthCapabilityCheckOutcome),
	}
}
