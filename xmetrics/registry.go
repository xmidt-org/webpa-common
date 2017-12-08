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
	NewCounterVec(string) *prometheus.CounterVec
	NewGaugeVec(string) *prometheus.GaugeVec
	NewHistogramVec(string) *prometheus.HistogramVec
	NewSummaryVec(string) *prometheus.SummaryVec
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

	namespace string
	subsystem string
	cache     map[string]prometheus.Collector
}

func (r *registry) NewCounterVec(name string) *prometheus.CounterVec {
	var counterVec *prometheus.CounterVec

	if existing, ok := r.cache[name]; ok {
		if counterVec, ok = existing.(*prometheus.CounterVec); !ok {
			panic(fmt.Errorf("The metric %s is not a counter", name))
		}
	} else {
		counterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		}, []string{})

		if err := r.Registry.Register(counterVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				counterVec = already.ExistingCollector.(*prometheus.CounterVec)
			} else {
				panic(err)
			}
		}

		r.cache[name] = counterVec
	}

	return counterVec
}

func (r *registry) NewCounter(name string) metrics.Counter {
	return gokitprometheus.NewCounter(r.NewCounterVec(name))
}

func (r *registry) NewGaugeVec(name string) *prometheus.GaugeVec {
	var gaugeVec *prometheus.GaugeVec

	if existing, ok := r.cache[name]; ok {
		if gaugeVec, ok = existing.(*prometheus.GaugeVec); !ok {
			panic(fmt.Errorf("The metric %s is not a gauge", name))
		}
	} else {
		gaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		}, []string{})

		if err := r.Registry.Register(gaugeVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				gaugeVec = already.ExistingCollector.(*prometheus.GaugeVec)
			} else {
				panic(err)
			}
		}

		r.cache[name] = gaugeVec
	}

	return gaugeVec
}

func (r *registry) NewGauge(name string) metrics.Gauge {
	return gokitprometheus.NewGauge(r.NewGaugeVec(name))
}

func (r *registry) NewHistogramVec(name string) *prometheus.HistogramVec {
	var histogramVec *prometheus.HistogramVec

	if existing, ok := r.cache[name]; ok {
		if histogramVec, ok = existing.(*prometheus.HistogramVec); !ok {
			panic(fmt.Errorf("The metric %s is not a histogram", name))
		}
	} else {
		histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		}, []string{})

		if err := r.Registry.Register(histogramVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				histogramVec = already.ExistingCollector.(*prometheus.HistogramVec)
			} else {
				panic(err)
			}
		}

		r.cache[name] = histogramVec
	}

	return histogramVec
}

func (r *registry) NewSummaryVec(name string) *prometheus.SummaryVec {
	var summaryVec *prometheus.SummaryVec

	if existing, ok := r.cache[name]; ok {
		if summaryVec, ok = existing.(*prometheus.SummaryVec); !ok {
			panic(fmt.Errorf("The metric %s is not a histogram", name))
		}
	} else {
		summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace: r.namespace,
			Subsystem: r.subsystem,
			Name:      name,
			Help:      name,
		}, []string{})

		if err := r.Registry.Register(summaryVec); err != nil {
			if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
				summaryVec = already.ExistingCollector.(*prometheus.SummaryVec)
			} else {
				panic(err)
			}
		}

		r.cache[name] = summaryVec
	}

	return summaryVec
}

// NewHistogram will return a Histogram for either a Summary or Histogram.  This is different
// behavior from metrics.Provider.
func (r *registry) NewHistogram(name string, _ int) metrics.Histogram {
	// we allow either a summary or a histogram to be wrapped as a go-kit Histogram
	if existing, ok := r.cache[name]; ok {
		switch vec := existing.(type) {
		case *prometheus.HistogramVec:
			return gokitprometheus.NewHistogram(vec)
		case *prometheus.SummaryVec:
			return gokitprometheus.NewSummary(vec)
		default:
			panic(fmt.Errorf("The metric %s is not a histogram or summary", name))
		}
	}

	return gokitprometheus.NewHistogram(r.NewHistogramVec(name))
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
		Registry:  pr,
		namespace: defaultNamespace,
		subsystem: defaultSubsystem,
		cache:     make(map[string]prometheus.Collector),
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

			r.cache[name] = counterVec

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

			r.cache[name] = gaugeVec

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

			r.cache[name] = histogramVec

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

			r.cache[name] = summaryVec

		default:
			return nil, fmt.Errorf("Unsupported metric type: %s", m.Type)
		}
	}

	return r, nil
}
