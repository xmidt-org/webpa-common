package xmetrics

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

const (
	CounterType   = "counter"
	GaugeType     = "gauge"
	HistogramType = "histogram"
	SummaryType   = "summary"
)

// Module is a function type that returns prebuilt metrics.
type Module func() []Metric

// Metric describes a single metric that will be preregistered.  This type loosely
// corresponds with Prometheus' Opts struct.  The fields in this type are the union
// of all necessary data for creating Prometheus metrics.
type Metric struct {
	// Name is the required name of this metric.  This value is required.
	Name string

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

	// ConstLabels are the Prometheus ConstLabels for this metric.  This field is optional.
	ConstLabels map[string]string

	// LabelNames are the Prometheus label names for this metric.  This field is optional.
	LabelNames []string

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

// NewCollector creates a Prometheus metric from a Metric descriptor.  The name must not be empty.
// If not supplied in the metric, namespace, subsystem, and help all take on defaults.
func NewCollector(m Metric) (prometheus.Collector, error) {
	if len(m.Name) == 0 {
		return nil, errors.New("A name is required for a metric")
	}

	var (
		namespace = m.Namespace
		subsystem = m.Subsystem
		help      = m.Help
	)

	if len(namespace) == 0 {
		namespace = DefaultNamespace
	}

	if len(subsystem) == 0 {
		subsystem = DefaultSubsystem
	}

	if len(help) == 0 {
		help = m.Name
	}

	switch m.Type {
	case CounterType:
		return prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        m.Name,
			Help:        help,
			ConstLabels: prometheus.Labels(m.ConstLabels),
		}, m.LabelNames), nil

	case GaugeType:
		return prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        m.Name,
			Help:        help,
			ConstLabels: prometheus.Labels(m.ConstLabels),
		}, m.LabelNames), nil

	case HistogramType:
		return prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        m.Name,
			Help:        help,
			Buckets:     m.Buckets,
			ConstLabels: prometheus.Labels(m.ConstLabels),
		}, m.LabelNames), nil

	case SummaryType:
		return prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        m.Name,
			Help:        help,
			Objectives:  m.Objectives,
			MaxAge:      m.MaxAge,
			AgeBuckets:  m.AgeBuckets,
			BufCap:      m.BufCap,
			ConstLabels: prometheus.Labels(m.ConstLabels),
		}, m.LabelNames), nil

	default:
		return nil, fmt.Errorf("Unsupported metric type: %s", m.Type)
	}
}

// Merger is the strategy for merging metrics from various sources.  It applies a configurable default
// namespace and subsystem to each metric.  This type implements a Fluent Interface which tracks the first
// error encountered.
type Merger struct {
	logger           log.Logger
	defaultNamespace string
	defaultSubsystem string
	namer            func(string, string, string) string
	merged           map[string]Metric
	err              error
}

// NewMerger creates a merging strategy with useful defaults.
func NewMerger() *Merger {
	return &Merger{
		logger:           logging.DefaultLogger(),
		defaultNamespace: DefaultNamespace,
		defaultSubsystem: DefaultSubsystem,
		namer:            prometheus.BuildFQName,
		merged:           make(map[string]Metric),
	}
}

// Logger sets a go-kit logger to use for merging output
func (mr *Merger) Logger(logger log.Logger) *Merger {
	if logger != nil {
		mr.logger = logger
	} else {
		mr.logger = logging.DefaultLogger()
	}

	return mr
}

// Namer sets the fully-qualified naming strategy for this merger.  This method applies to all subsequent
// AddXXX calls, but does not affect any metrics already merged.  If f is nil, an internal default naming
// strategy is used.
func (mr *Merger) Namer(f func(namespace, subsystem, name string) string) *Merger {
	if f == nil {
		mr.namer = prometheus.BuildFQName
	} else {
		mr.namer = f
	}

	return mr
}

// DefaultNamespace sets the default namespace used for metrics that do not specify one.  This value applies
// to all subsequent AddXXX calls, but does not affect any metrics already merged.  If the value is empty,
// the global DefaultNamespace constant is used.
func (mr *Merger) DefaultNamespace(v string) *Merger {
	if len(v) == 0 {
		mr.defaultNamespace = DefaultNamespace
	} else {
		mr.defaultNamespace = v
	}

	return mr
}

// DefaultSubsystem sets the default subsystem used for metrics that do not specify one.  This value applies
// to all subsequent AddXXX calls, but does not affect any metrics already merged.  If the value is empty,
// the global DefaultSubsystem constant is used.
func (mr *Merger) DefaultSubsystem(v string) *Merger {
	if len(v) == 0 {
		mr.defaultSubsystem = DefaultSubsystem
	} else {
		mr.defaultSubsystem = v
	}

	return mr
}

// Merged returns the built map of metrics from all sources, keyed by fully-qualified name
func (mr *Merger) Merged() map[string]Metric {
	return mr.merged
}

// Err returns any error that occurred during merging.  When this method returns non-nil,
// no further additions will be accepted.
func (mr *Merger) Err() error {
	return mr.err
}

func (mr *Merger) tryAdd(allowOverride bool, m Metric) bool {
	if mr.err != nil {
		return false
	}

	defer func() {
		if mr.err != nil {
			mr.logger.Log(
				level.Key(), level.ErrorValue(),
				logging.MessageKey(), "failed to merge metrics",
				logging.ErrorKey(), mr.err,
				"name", m.Name,
				"namespace", m.Namespace,
				"subsystem", m.Subsystem,
				"type", m.Type,
			)
		}
	}()

	if len(m.Name) == 0 {
		mr.err = errors.New("names are required for metrics")
		return false
	}

	if len(m.Namespace) == 0 {
		m.Namespace = mr.defaultNamespace
	}

	if len(m.Subsystem) == 0 {
		m.Subsystem = mr.defaultSubsystem
	}

	fqn := mr.namer(m.Namespace, m.Subsystem, m.Name)
	mr.logger.Log(
		level.Key(), level.DebugValue(),
		logging.MessageKey(), "merging metric",
		"name", m.Name,
		"namespace", m.Namespace,
		"subsystem", m.Subsystem,
		"fqn", fqn,
		"type", m.Type,
	)

	if existing, ok := mr.merged[fqn]; ok {
		if !allowOverride {
			mr.err = fmt.Errorf("duplicate metric with name: %s", fqn)
			return false
		}

		// we never allow a metric to override one of a different type
		if existing.Type != m.Type {
			mr.err = fmt.Errorf("metric %s was expected to be of type %s, but was of type %s", fqn, existing.Type, m.Type)
			return false
		}
	}

	mr.merged[fqn] = m
	return true
}

// AddMetrics merges the given slice of metrics into this instance.  If Err() returns non-nil, this method
// has no effect.
//
// If allowOverride is false, then any metric with the same fully-qualified name as a metric already merged
// will result in an error.  If allowOverride is true, then metrics with the same name are allowed to override
// previously merged metrics if and only if they are of the same type.
func (mr *Merger) AddMetrics(allowOverride bool, m []Metric) *Merger {
	for _, e := range m {
		if !mr.tryAdd(allowOverride, e) {
			break
		}
	}

	return mr
}

// AddModules merges zero or more modules into this instance.  If Err() returns non-nil, this method
// has no effect.
//
// See AddMetrics for a description of allowOverride.
func (mr *Merger) AddModules(allowOverride bool, m ...Module) *Merger {
	for _, mf := range m {
		for _, e := range mf() {
			if !mr.tryAdd(allowOverride, e) {
				return mr
			}
		}
	}

	return mr
}
