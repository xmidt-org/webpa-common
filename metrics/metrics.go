package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

//Provider is intended to be a metrics API for webpa servers.
type Provider struct {
	DefaultGathererInUse bool
}

//GetCounter returns a Counter metrics Collector
//If MetricsTool.DefaultGathererInUse is true, it will go ahead and do the registration for such metric
//with the prometheus defaultGatherer
func (m *Provider) GetCounter(name, help string, labelValues []string) (counter metrics.Counter) {
	opts := stdprometheus.CounterOpts{Name: name, Help: help}
	if m.DefaultGathererInUse {
		counter = prometheus.NewCounterFrom(opts, labelValues) // registers with defaultGatherer
	} else {
		counter = prometheus.NewCounter(stdprometheus.NewCounterVec(opts, labelValues))
	}
	return
}
