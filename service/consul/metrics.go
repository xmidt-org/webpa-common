// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package consul

import (
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	metrics := []xmetrics.Metric{
		{
			Name:       chrysom.PollCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			LabelNames: []string{chrysom.OutcomeLabel},
		},
	}
	return metrics
}
