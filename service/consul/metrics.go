/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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
