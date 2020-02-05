package basculechecks

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

//Names for our metrics
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
	NoCapabilitiesMatch      = "no_capabilities_match"
)

//Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name:       AuthCapabilityCheckOutcome,
			Type:       xmetrics.CounterType,
			Help:       "Counter for success and failure reason results through bascule",
			LabelNames: []string{OutcomeLabel, ReasonLabel, ClientIDLabel, PartnerIDLabel, EndpointLabel},
		},
	}
}

//AuthCapabilityCheckMeasures describes the defined metrics that will be used by clients
type AuthCapabilityCheckMeasures struct {
	CapabilityCheckOutcome metrics.Counter
}

//NewAuthCapabilityCheckMeasures realizes desired metrics
func NewAuthCapabilityCheckMeasures(p provider.Provider) *AuthCapabilityCheckMeasures {
	return &AuthCapabilityCheckMeasures{
		CapabilityCheckOutcome: p.NewCounter(AuthCapabilityCheckOutcome),
	}
}
