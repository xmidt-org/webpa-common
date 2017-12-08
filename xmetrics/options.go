package xmetrics

import "time"

const (
	DefaultNamespace = "global"
	DefaultSubsystem = "development"
)

// Metric describes a single metric that will be preregistered.  This type loosely
// corresponds with Prometheus' Opts struct.
type Metric struct {
	// Type is the required type of metric.  This value must be one of the constants defined in this package.
	Type string

	// Namespace is the namespace of this metric.  This value is optional.  The enclosing Options' Namespace
	// field is used if this is not supplied.
	Namespace string

	// Subsystem is the subsystem of this metric.  This value is optional.  The enclosing Options' Subsystem
	// field is used if this is not supplied.
	Subsystem string

	// Help is the help string for this metric.  If not supplied, the metric's name is used
	Help string

	// Labels are the Prometheus ConstLabels for this metric.  This field is optional.
	Labels map[string]string

	// Buckets describes the observation buckets for a histogram.  This field is only valid for histogram metrics
	// and is ignored for other metric types.
	Buckets []float64

	// Objectives is the Summary objectives.  This field is only valid for summary metrics, and is ignored
	// for other metric types.
	Objectives map[float64]float64

	// MaxAge is the Summary MaxAge.  This field is only valid for summary metrics, and is ignored
	// for other metric types.
	MaxAge time.Duration

	// AgeBuckets is the Summary AgeBuckets.  This field is only valid for summary metrics, and is ignored
	// for other metric types.
	AgeBuckets uint32

	// BufCap is the Summary BufCap.  This field is only valid for summary metrics, and is ignored
	// for other metric types.
	BufCap uint32
}

// Options is the configurable options for creating a Prometheus registry
type Options struct {
	// Namespace is the global default namespace for metrics which don't define a namespace (or for ad hoc metrics).
	// If not supplied, DefaultNamespace is used.
	Namespace string

	// Subsystem is the global default subsystem for metrics which don't define a subsystem (or for ad hoc metrics).
	// If not supplied, DefaultSubsystem is used.
	Subsystem string

	// Pedantic indicates whether the registry is created via NewPedanticRegistry().  By default, this is false.  Set
	// to true for testing or development.
	Pedantic bool

	// Metrics defines the map of predefined metrics.  These metrics will be defined immediately by an Registry
	// created using this Options instance.  This field is optional.
	Metrics map[string]Metric
}

func (o *Options) namespace() string {
	if o != nil && len(o.Namespace) > 0 {
		return o.Namespace
	}

	return DefaultNamespace
}

func (o *Options) subsystem() string {
	if o != nil && len(o.Subsystem) > 0 {
		return o.Subsystem
	}

	return DefaultSubsystem
}

func (o *Options) pedantic() bool {
	if o != nil {
		return o.Pedantic
	}

	return false
}

func (o *Options) metrics() map[string]Metric {
	if o != nil {
		return o.Metrics
	}

	return nil
}
