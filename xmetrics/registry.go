package xmetrics

import (
	"fmt"

	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusProvider is a Prometheus-specific version of go-kit's metrics.Provider.  Use this interface
// when interacting directly with Prometheus.
type PrometheusProvider interface {
	NewCounterVec(name string) *prometheus.CounterVec
	NewCounterVecEx(namespace, subsystem, name string) *prometheus.CounterVec

	NewGaugeVec(name string) *prometheus.GaugeVec
	NewGaugeVecEx(namespace, subsystem, name string) *prometheus.GaugeVec

	NewHistogramVec(name string) *prometheus.HistogramVec
	NewHistogramVecEx(namespace, subsystem, name string) *prometheus.HistogramVec

	NewSummaryVec(name string) *prometheus.SummaryVec
	NewSummaryVecEx(namespace, subsystem, name string) *prometheus.SummaryVec
}

// Registry is the core abstraction for this package.  It is a Prometheus gatherer and a go-kit metrics.Provider all in one.
//
// The Provider implementation works slightly differently than the go-kit implementation.  For any metric that is already defined
// the provider returns a new go-kit wrapper for that metric.  Additionally, new metrics (including ad hoc metrics) are cached
// and returned by subsequent calles to the Provider methods.
type Registry interface {
	PrometheusProvider
	provider.Provider
	prometheus.Gatherer
}

// registry is the internal Registry implementation
type registry struct {
	prometheus.Gatherer
	prometheus.Registerer

	namespace     string
	subsystem     string
	preregistered map[string]prometheus.Collector
}

func (r *registry) NewCounterVec(name string) *prometheus.CounterVec {
	return r.NewCounterVecEx(r.namespace, r.subsystem, name)
}

func (r *registry) NewCounterVecEx(namespace, subsystem, name string) *prometheus.CounterVec {
	key := prometheus.BuildFQName(namespace, subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		if counterVec, ok := existing.(*prometheus.CounterVec); ok {
			return counterVec
		}

		panic(fmt.Errorf("The preregistered metric %s is not a counter", key))
	}

	counterVec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	)

	if err := r.Register(counterVec); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return already.ExistingCollector.(*prometheus.CounterVec)
		} else {
			panic(err)
		}
	}

	return counterVec
}

func (r *registry) NewCounter(name string) metrics.Counter {
	return gokitprometheus.NewCounter(r.NewCounterVec(name))
}

func (r *registry) NewGaugeVec(name string) *prometheus.GaugeVec {
	return r.NewGaugeVecEx(r.namespace, r.subsystem, name)
}

func (r *registry) NewGaugeVecEx(namespace, subsystem, name string) *prometheus.GaugeVec {
	key := prometheus.BuildFQName(namespace, subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		if gaugeVec, ok := existing.(*prometheus.GaugeVec); ok {
			return gaugeVec
		}

		panic(fmt.Errorf("The preregistered metric %s is not a gauge", key))
	}

	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	)

	if err := r.Register(gaugeVec); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return already.ExistingCollector.(*prometheus.GaugeVec)
		} else {
			panic(err)
		}
	}

	return gaugeVec
}

func (r *registry) NewGauge(name string) metrics.Gauge {
	return gokitprometheus.NewGauge(r.NewGaugeVec(name))
}

func (r *registry) NewHistogramVec(name string) *prometheus.HistogramVec {
	return r.NewHistogramVecEx(r.namespace, r.subsystem, name)
}

func (r *registry) NewHistogramVecEx(namespace, subsystem, name string) *prometheus.HistogramVec {
	key := prometheus.BuildFQName(namespace, subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		if histogramVec, ok := existing.(*prometheus.HistogramVec); ok {
			return histogramVec
		}

		panic(fmt.Errorf("The preregistered metric %s is not a histogram", key))
	}

	histogramVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	)

	if err := r.Register(histogramVec); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return already.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			panic(err)
		}
	}

	return histogramVec
}

// NewHistogram has some special logic over and above the go-kit implementations.  This method allows a summary or
// a histogram as the underlying metric for the go-kit metrics.Histogram.
func (r *registry) NewHistogram(name string, _ int) metrics.Histogram {
	key := prometheus.BuildFQName(r.namespace, r.subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		switch e := existing.(type) {
		case *prometheus.HistogramVec:
			return gokitprometheus.NewHistogram(e)
		case *prometheus.SummaryVec:
			return gokitprometheus.NewSummary(e)
		default:
			panic(fmt.Errorf("The preregistered metric %s is not a histogram or a summary", key))
		}
	}

	return gokitprometheus.NewHistogram(r.NewHistogramVec(name))
}

func (r *registry) NewSummaryVec(name string) *prometheus.SummaryVec {
	return r.NewSummaryVecEx(r.namespace, r.subsystem, name)
}

func (r *registry) NewSummaryVecEx(namespace, subsystem, name string) *prometheus.SummaryVec {
	key := prometheus.BuildFQName(namespace, subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		if summaryVec, ok := existing.(*prometheus.SummaryVec); ok {
			return summaryVec
		}

		panic(fmt.Errorf("The preregistered metric %s is not a histogram", key))
	}

	summaryVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	)

	if err := r.Register(summaryVec); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return already.ExistingCollector.(*prometheus.SummaryVec)
		} else {
			panic(err)
		}
	}

	return summaryVec
}

// Stop is just here to implement metrics.Provider.  This method is a noop.
func (r *registry) Stop() {
}

// NewRegistry creates an xmetrics.Registry from an externally supplied set of Options and a set
// of modules, which are functions that just return Metrics to register.  The module functions are
// expected to come from application or library code, and are to define any built-in metrics.  Metrics
// present in the options will override any corresponding metric from modules.
func NewRegistry(o *Options, modules ...Module) (Registry, error) {
	// merge all the metrics, allowing options to override modules
	merger := NewMerger().
		Logger(o.logger()).
		DefaultNamespace(o.namespace()).
		DefaultSubsystem(o.subsystem()).
		AddModules(false, modules...).
		AddModules(true, o.Module)

	if merger.Err() != nil {
		return nil, merger.Err()
	}

	var (
		pr = o.registry()
		r  = &registry{
			Registerer:    pr,
			Gatherer:      pr,
			namespace:     o.namespace(),
			subsystem:     o.subsystem(),
			preregistered: make(map[string]prometheus.Collector),
		}
	)

	for name, metric := range merger.Merged() {
		// merged metrics will have namespace and subsystem set appropriately
		c, err := NewCollector(metric)
		if err != nil {
			return nil, err
		}

		r.preregistered[name] = c
	}

	return r, nil
}

// MustNewRegistry is like NewRegistry, except that it panics when NewRegistry would return an error.
func MustNewRegistry(o *Options, modules ...Module) Registry {
	r, err := NewRegistry(o, modules...)
	if err != nil {
		panic(err)
	}

	return r
}
