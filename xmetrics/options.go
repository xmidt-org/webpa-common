package xmetrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	DefaultNamespace = "test"
	DefaultSubsystem = "test"
)

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

	// DisableGoCollector controls whether the Go Collector is registered with the Registry.  By default this is false,
	// meaning that a GoCollector is registered.
	DisableGoCollector bool

	// DisableProcessCollector controls whether the Process Collector is registered with the Registry.  By default this is false,
	// meaning that a ProcessCollector is registered.
	DisableProcessCollector bool

	// Metrics defines the set of predefined metrics.  These metrics will be defined immediately by an Registry
	// created using this Options instance.  This field is optional.
	//
	// Any duplicate metrics will cause an error.  Duplicate metrics are defined as those having the same namespace,
	// subsystem, and name.
	Metrics []Metric
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

func (o *Options) registry() *prometheus.Registry {
	var pr *prometheus.Registry

	if o.pedantic() {
		pr = prometheus.NewPedanticRegistry()
	} else {
		pr = prometheus.NewRegistry()
	}

	if !o.disableGoCollector() {
		pr.MustRegister(prometheus.NewGoCollector())
	}

	if !o.disableProcessCollector() {
		pr.MustRegister(prometheus.NewProcessCollector(os.Getpid(), o.namespace()))
	}

	return pr
}

func (o *Options) disableGoCollector() bool {
	if o != nil {
		return o.DisableGoCollector
	}

	return false
}

func (o *Options) disableProcessCollector() bool {
	if o != nil {
		return o.DisableProcessCollector
	}

	return false
}

func (o *Options) metrics() []Metric {
	if o != nil {
		return o.Metrics
	}

	return nil
}
