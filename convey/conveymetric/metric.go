// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package conveymetric

import (
	"github.com/go-kit/kit/metrics"
	"github.com/xmidt-org/webpa-common/v2/convey"
)

// UnknownLabelValue is a constant for when key/tag can not be found in the C JSON.
const UnknownLabelValue = "unknown"

// Closure will be returned after Update(), this should be used to update the struct, aka decrement the count.
type Closure func()

// TagLabelPair is a convenient structure for inputs to create a new convey metric.
type TagLabelPair struct {
	Tag   string
	Label string
}

// Interface provides a way of updating an internal resource.
type Interface interface {
	// Update takes the convey JSON to update internal struct, and return a closure to update the struct again, or an
	// error
	//
	// Note: Closure should only be called once.
	Update(data convey.C, labelPairs ...string) (Closure, error)
}

// NewConveyMetric produces an Interface where gauge is the internal structure to update, tag is the key in the C JSON
// to update the gauge, and label is the `key` for the gauge cardinality.
//
// Note: The Gauge must have the label as one of the constant labels, (aka. the name of the gauge)
func NewConveyMetric(gauge metrics.Gauge, pairs ...TagLabelPair) Interface {
	return &cMetric{
		pairs: pairs,
		gauge: gauge,
	}
}

// cMetric is the internal Interface implementation
type cMetric struct {
	pairs []TagLabelPair
	gauge metrics.Gauge
}

func (m *cMetric) Update(data convey.C, baseLabelPairs ...string) (Closure, error) {
	labelPairs := baseLabelPairs
	for _, pair := range m.pairs {
		labelValue := UnknownLabelValue
		if item, ok := data[pair.Tag].(string); ok {
			labelValue = item
		}
		labelPairs = append(labelPairs, pair.Label, labelValue)
	}
	m.gauge.With(labelPairs...).Add(1.0)
	return func() { m.gauge.With(labelPairs...).Add(-1.0) }, nil
}
