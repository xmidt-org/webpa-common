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
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/themis/xlog"
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
	ServerLabel    = "server"
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

// help messages
const (
	capabilityCheckHelpMsg =       "Counter for the capability checker, providing outcome information by client, partner, and endpoint",


)

// Metrics returns the Metrics relevant to this package targeting our older non uber/fx applications.
// To initialize the metrics, use NewAuthCapabilityCheckMeasures().
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       AuthCapabilityCheckOutcome,
			Type:       xmetrics.CounterType,
			Help:       capabilityCheckHelpMsg,
			LabelNames: []string{OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel},
		},
	}
}

// ProvideMetrics provides the metrics relevant to this package as uber/fx options.
// This is now deprecated in favor of ProvideMetricsVec.
func ProvideMetrics() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounter(prometheus.CounterOpts{
			Name:        AuthCapabilityCheckOutcome,
			Help:        capabilityCheckHelpMsg,
			ConstLabels: nil,
		}, OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel),
	)
}

// ProvideMetricsVec provides the metrics relevant to this package as uber/fx options.
// The provided metrics are prometheus vectors which gives access to more advanced operations such as CurryWith(labels).
func ProvideMetricsVec() fx.Option {
	return fx.Provide(
		themisXmetrics.ProvideCounterVec(prometheus.CounterOpts{
			Name:        AuthCapabilityCheckOutcome,
			Help:        capabilityCheckHelpMsg,
			ConstLabels: nil,
		}, ServerLabel, OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel),
	)
}

// AuthCapabilityCheckMeasures describes the defined metrics that will be used by clients
type AuthCapabilityCheckMeasures struct {
	CapabilityCheckOutcome metrics.Counter
}

// NewAuthCapabilityCheckMeasures realizes desired metrics. It's intended to be used alongside Metrics() for
// our older non uber/fx applications.
func NewAuthCapabilityCheckMeasures(p provider.Provider) *AuthCapabilityCheckMeasures {
	return &AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: p.NewCounter(AuthCapabilityCheckOutcome),
	}
}

// BaseMeasuresIn is an uber/fx parameter with base metrics ready to be curried into child metrics based on
// custom labels.
type BaseMeasuresIn struct {
	fx.In
	Logger                 log.Logger
	CapabilityCheckOutcome *prometheus.CounterVec `name:"auth_capability_check"`
}

// MeasuresFactory facilitates the creation of child metrics based on server labels.
type MeasuresFactory struct {
	ServerName string
}

// NewMeasures builds the metric listener from the provided raw metrics.
func (m MeasuresFactory) NewMeasures(in BaseMeasuresIn) (*AuthCapabilityCheckMeasures, error) {
	capabilityCheckOutcomeCounterVec, err := in.CapabilityCheckOutcome.CurryWith(prometheus.Labels{ServerLabel: m.ServerName})
	if err != nil {
		return nil, err
	}
	in.Logger.Log(level.Key(), level.DebugValue(), xlog.MessageKey(), "building auth capability measures", ServerLabel, m.ServerName)
	return &AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: gokitprometheus.NewCounter(capabilityCheckOutcomeCounterVec),
	}, nil
}

// Annotated provides the measures as an annotated component with the name "[SERVER]_capability_measures"
func (m MeasuresFactory) Annotated() fx.Annotated {
	return fx.Annotated{
		Name:   fmt.Sprintf("%s_capability_measures", m.ServerName),
		Target: m.NewMeasures,
	}
}
