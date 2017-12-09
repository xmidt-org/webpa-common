package xmetrics

import (
	"errors"
	"fmt"

	"github.com/go-kit/kit/metrics"
	gokitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	CounterType   = "counter"
	GaugeType     = "gauge"
	HistogramType = "histogram"
	SummaryType   = "summary"
)

// PrometheusProvider is a Prometheus-specific version of go-kit's metrics.Provider.  Use this interface
// when interacting directly with Prometheus.
type PrometheusProvider interface {
	// NewCounterVec creates a new Prometheus CounterVec with the extra functionality of supplying default
	// namespace, subsystem, and help using this instance's configuration.  If the options refer to a preregistered
	// counter, the label names are ignored and that counter is returned.
	NewCounterVec(prometheus.CounterOpts, []string) *prometheus.CounterVec

	// NewGaugeVec creates a new Prometheus GaugeVec with the extra functionality of supplying default
	// namespace, subsystem, and help using this instance's configuration.  If the options refer to a preregistered
	// gauge, the label names are ignored and that gauge is returned.
	NewGaugeVec(prometheus.GaugeOpts, []string) *prometheus.GaugeVec

	// NewHistogramVec creates a new Prometheus HistogramVec with the extra functionality of supplying default
	// namespace, subsystem, and help using this instance's configuration.  If the options refer to a preregistered
	// histogram, the label names are ignored and that histogram is returned.
	NewHistogramVec(prometheus.HistogramOpts, []string) *prometheus.HistogramVec

	// NewSummaryVec creates a new Prometheus SummaryVec with the extra functionality of supplying default
	// namespace, subsystem, and help using this instance's configuration.  If the options refer to a preregistered
	// summary, the label names are ignored and that summary is returned.
	NewSummaryVec(prometheus.SummaryOpts, []string) *prometheus.SummaryVec
}

// Registry is the core abstraction for this package.  It is a Prometheus registry and a go-kit metrics.Provider all in one.
//
// The Provider implementation works slightly differently than the go-kit implementation.  For any metric that is already defined
// the provider returns a new go-kit wrapper for that metric.  Additionally, new metrics (including ad hoc metrics) are cached
// and returned by subsequent calles to the Provider methods.
type Registry interface {
	PrometheusProvider
	provider.Provider
	prometheus.Gatherer
	prometheus.Registerer
}

// registry is the internal Registry implementation
type registry struct {
	*prometheus.Registry

	namespace     string
	subsystem     string
	preregistered map[string]prometheus.Collector
}

// keyFor computes the unique key for a metric using the same logic as Prometheus.
func keyFor(namespace, subsystem, name string) string {
	return namespace + "_" + subsystem + "_" + name
}

func (r *registry) NewCounterVec(opts prometheus.CounterOpts, labelNames []string) *prometheus.CounterVec {
	if len(opts.Name) == 0 {
		panic("A name is required")
	}

	if len(opts.Help) == 0 {
		opts.Help = opts.Name
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = r.namespace
	}

	if len(opts.Subsystem) == 0 {
		opts.Subsystem = r.subsystem
	}

	var (
		counterVec *prometheus.CounterVec
		key        = keyFor(opts.Namespace, opts.Subsystem, opts.Name)
	)

	if existing, ok := r.preregistered[key]; ok {
		if counterVec, ok = existing.(*prometheus.CounterVec); !ok {
			panic(fmt.Errorf("The preregistered metric %s is not a counter", key))
		}
	} else {
		counterVec = prometheus.NewCounterVec(opts, labelNames)
		if err := r.Registry.Register(counterVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				counterVec = already.ExistingCollector.(*prometheus.CounterVec)
			} else {
				panic(err)
			}
		}
	}

	return counterVec
}

func (r *registry) NewCounter(name string) metrics.Counter {
	return gokitprometheus.NewCounter(r.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	))
}

func (r *registry) NewGaugeVec(opts prometheus.GaugeOpts, labelNames []string) *prometheus.GaugeVec {
	if len(opts.Name) == 0 {
		panic("A name is required")
	}

	if len(opts.Help) == 0 {
		opts.Help = opts.Name
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = r.namespace
	}

	if len(opts.Subsystem) == 0 {
		opts.Subsystem = r.subsystem
	}

	var (
		gaugeVec *prometheus.GaugeVec
		key      = keyFor(opts.Namespace, opts.Subsystem, opts.Name)
	)

	if existing, ok := r.preregistered[key]; ok {
		if gaugeVec, ok = existing.(*prometheus.GaugeVec); !ok {
			panic(fmt.Errorf("The preregistered metric %s is not a gauge", key))
		}
	} else {
		gaugeVec = prometheus.NewGaugeVec(opts, labelNames)
		if err := r.Registry.Register(gaugeVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				gaugeVec = already.ExistingCollector.(*prometheus.GaugeVec)
			} else {
				panic(err)
			}
		}
	}

	return gaugeVec
}

func (r *registry) NewGauge(name string) metrics.Gauge {
	return gokitprometheus.NewGauge(r.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	))
}

func (r *registry) NewHistogramVec(opts prometheus.HistogramOpts, labelNames []string) *prometheus.HistogramVec {
	if len(opts.Name) == 0 {
		panic("A name is required")
	}

	if len(opts.Help) == 0 {
		opts.Help = opts.Name
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = r.namespace
	}

	if len(opts.Subsystem) == 0 {
		opts.Subsystem = r.subsystem
	}

	var (
		histogramVec *prometheus.HistogramVec
		key          = keyFor(opts.Namespace, opts.Subsystem, opts.Name)
	)

	if existing, ok := r.preregistered[key]; ok {
		if histogramVec, ok = existing.(*prometheus.HistogramVec); !ok {
			panic(fmt.Errorf("The preregistered metric %s is not a histogram", key))
		}
	} else {
		histogramVec = prometheus.NewHistogramVec(opts, labelNames)
		if err := r.Registry.Register(histogramVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				histogramVec = already.ExistingCollector.(*prometheus.HistogramVec)
			} else {
				panic(err)
			}
		}
	}

	return histogramVec
}

func (r *registry) NewSummaryVec(opts prometheus.SummaryOpts, labelNames []string) *prometheus.SummaryVec {
	if len(opts.Name) == 0 {
		panic("A name is required")
	}

	if len(opts.Help) == 0 {
		opts.Help = opts.Name
	}

	if len(opts.Namespace) == 0 {
		opts.Namespace = r.namespace
	}

	if len(opts.Subsystem) == 0 {
		opts.Subsystem = r.subsystem
	}

	var (
		summaryVec *prometheus.SummaryVec
		key        = keyFor(opts.Namespace, opts.Subsystem, opts.Name)
	)

	if existing, ok := r.preregistered[key]; ok {
		if summaryVec, ok = existing.(*prometheus.SummaryVec); !ok {
			panic(fmt.Errorf("The preregistered metric %s is not a histogram", key))
		}
	} else {
		summaryVec = prometheus.NewSummaryVec(opts, labelNames)
		if err := r.Registry.Register(summaryVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				summaryVec = already.ExistingCollector.(*prometheus.SummaryVec)
			} else {
				panic(err)
			}
		}
	}

	return summaryVec
}

// NewHistogram will return a Histogram for either a Summary or Histogram.  This is different
// behavior from metrics.Provider.
func (r *registry) NewHistogram(name string, _ int) metrics.Histogram {
	// we allow either a summary or a histogram to be wrapped as a go-kit Histogram
	key := keyFor(r.namespace, r.subsystem, name)
	if existing, ok := r.preregistered[key]; ok {
		switch vec := existing.(type) {
		case *prometheus.HistogramVec:
			return gokitprometheus.NewHistogram(vec)
		case *prometheus.SummaryVec:
			return gokitprometheus.NewSummary(vec)
		default:
			panic(fmt.Errorf("The preregistered metric %s is not a histogram or summary", name))
		}
	}

	return gokitprometheus.NewHistogram(r.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		},
		[]string{},
	))
}

func (r *registry) Stop() {
}

func NewRegistry(o *Options) (Registry, error) {
	var (
		defaultNamespace = o.namespace()
		defaultSubsystem = o.subsystem()
		pr               *prometheus.Registry
	)

	if o.pedantic() {
		pr = prometheus.NewPedanticRegistry()
	} else {
		pr = prometheus.NewRegistry()
	}

	r := &registry{
		Registry:      pr,
		namespace:     defaultNamespace,
		subsystem:     defaultSubsystem,
		preregistered: make(map[string]prometheus.Collector),
	}

	for name, m := range o.metrics() {
		if len(name) == 0 {
			return nil, errors.New("Metric names cannot be empty")
		}

		var (
			namespace = m.Namespace
			subsystem = m.Subsystem
			help      = m.Help
		)

		if len(namespace) == 0 {
			namespace = defaultNamespace
		}

		if len(subsystem) == 0 {
			subsystem = defaultSubsystem
		}

		if len(help) == 0 {
			help = name
		}

		key := keyFor(namespace, subsystem, name)

		switch m.Type {
		case CounterType:
			counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				ConstLabels: prometheus.Labels(m.Labels),
			}, []string{})

			if err := r.Registry.Register(counterVec); err != nil {
				return nil, fmt.Errorf("Error while preregistering metric %s: %s", name, err)
			}

			r.preregistered[key] = counterVec

		case GaugeType:
			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				ConstLabels: prometheus.Labels(m.Labels),
			}, []string{})

			if err := r.Registry.Register(gaugeVec); err != nil {
				return nil, fmt.Errorf("Error while preregistering metric %s: %s", name, err)
			}

			r.preregistered[key] = gaugeVec

		case HistogramType:
			histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				Buckets:     m.Buckets,
				ConstLabels: prometheus.Labels(m.Labels),
			}, []string{})

			if err := r.Registry.Register(histogramVec); err != nil {
				return nil, fmt.Errorf("Error while preregistering metric %s: %s", name, err)
			}

			r.preregistered[key] = histogramVec

		case SummaryType:
			summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
				Namespace:   namespace,
				Subsystem:   subsystem,
				Name:        name,
				Help:        help,
				Objectives:  m.Objectives,
				MaxAge:      m.MaxAge,
				AgeBuckets:  m.AgeBuckets,
				BufCap:      m.BufCap,
				ConstLabels: prometheus.Labels(m.Labels),
			}, []string{})

			if err := r.Registry.Register(summaryVec); err != nil {
				return nil, fmt.Errorf("Error while preregistering metric %s: %s", name, err)
			}

			r.preregistered[key] = summaryVec

		default:
			return nil, fmt.Errorf("Unsupported metric type: %s", m.Type)
		}
	}

	return r, nil
}
