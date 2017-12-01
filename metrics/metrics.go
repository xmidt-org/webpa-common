package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type MetricsTool struct {
	DefaultGathererInUse bool
}

func (m *MetricsTool) GetCounter(name, help string) (counter metrics.Counter) {
	//todo need to account for
	if m.DefaultGathererInUse {
		counter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{Name: name, Help: help}, []string{})
	} else {
		//todo
	}
	return
}
