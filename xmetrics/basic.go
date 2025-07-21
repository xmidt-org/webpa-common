// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xmetrics

// Adder represents a metrics to which deltas can be added.  Go-kit's metrics.Counter, metrics.Gauge, and
// several prometheus interfaces implement this interface.
type Adder interface {
	Add(float64)
}

// Incrementer represents an Adder which can only be incremented by 1
type Incrementer interface {
	Inc()
}

type incrementerAdapter struct {
	Adder
}

func (ia incrementerAdapter) Inc() {
	ia.Add(1.0)
}

// NewIncrementer creates a wrapper around a given Adder.  This is syntactic sugar around using the Adder directly.
func NewIncrementer(a Adder) Incrementer {
	return incrementerAdapter{a}
}

// Setter represents a metric that can receive updates, e.g. a gauge.  Go-kit's metrics.Gauge
// and prometheus gauges implement this interface.
type Setter interface {
	Set(float64)
}

// AddSetter represents a metric that can both have deltas applied and receive new values.  Gauges most
// commonly implement this interface.
type AddSetter interface {
	Adder
	Setter
}

// Observer is a type of metric which receives observations.  Histograms and summaries implement this interface.
type Observer interface {
	Observe(float64)
}

// Valuer is implemented by metrics which can expose their current value.  A couple of go-kit's metrics/generic types implement this interface.
type Valuer interface {
	Value() float64
}

// LabelValuer is implemented by metrics which expose what their label values are.
// All of go-kit's metrics/generic types implement this interface.
type LabelValuer interface {
	LabelValues() []string
}
