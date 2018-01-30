package xmetrics

// Adder represents a metrics to which deltas can be added.  Go-kit's metrics.Counter, metrics.Gauge, and
// several prometheus interfaces implement this interface.
type Adder interface {
	Add(float64)
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
