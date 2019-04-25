package client

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	OutboundInFlightGauge         = "outbound_inflight"
	OutboundRequestDuration       = "outbound_request_duration_seconds"
	OutboundRequestCounter        = "outbound_requests"
	OutboundDroppedMessageCounter = "outbound_dropped_messages"
	OutboundRetries               = "outbound_retries"
)

func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		xmetrics.Metric{
			Name: OutboundInFlightGauge,
			Type: "gauge",
			Help: "The number of active, in-flight requests from devices",
		},
		xmetrics.Metric{
			Name:    OutboundRequestDuration,
			Type:    "histogram",
			Help:    "The durations of outbound requests from devices",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		xmetrics.Metric{
			Name:       OutboundRequestCounter,
			Type:       "counter",
			Help:       "The count of outbound requests",
			LabelNames: []string{"code"},
		},
		xmetrics.Metric{
			Name: OutboundDroppedMessageCounter,
			Type: "counter",
			Help: "The total count of messages dropped due to a full outbound queue",
		},
		xmetrics.Metric{
			Name: OutboundRetries,
			Type: "counter",
			Help: "The total count of HTTP retries",
		},
	}
}

type OutboundMetricOptions struct {
	InFlight        bool
	RequestDuration bool
	RequestCounter  bool
	DroppedMessages bool
	OutboundRetries bool
}

type OutboundMeasures struct {
	InFlight        prometheus.Gauge
	RequestDuration prometheus.Observer
	RequestCounter  *prometheus.CounterVec
	DroppedMessages *prometheus.CounterVec
	Retries         metrics.Counter
}

func NewOutboundMeasures(r xmetrics.Registry) OutboundMeasures {
	return OutboundMeasures{
		InFlight:        r.NewGaugeVec(OutboundInFlightGauge).WithLabelValues(),
		RequestDuration: r.NewHistogramVec(OutboundRequestDuration).WithLabelValues(),
		RequestCounter:  r.NewCounterVec(OutboundRequestCounter),
		DroppedMessages: r.NewCounterVec(OutboundDroppedMessageCounter),
		Retries:         r.NewCounter(OutboundRetries),
	}
}

func InstrumentOutboundDuration(obs prometheus.Observer, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		start := time.Now()
		response, err := next.RoundTrip(request)
		if err == nil {
			obs.Observe(time.Since(start).Seconds())
		}

		return response, err
	})
}

func InstrumentOutboundCounter(counter *prometheus.CounterVec, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		response, err := next.RoundTrip(request)
		if err == nil {
			// use "200" as the result from a 0 or negative status code, to be consistent with other golang APIs
			labels := prometheus.Labels{"code": "200"}
			if response.StatusCode > 0 {
				labels["code"] = strconv.Itoa(response.StatusCode)
			}

			counter.With(labels).Inc()
		}

		return response, err
	})
}

func InstrumentOutboundDroppedMessages(counter *prometheus.CounterVec, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		response, err := next.RoundTrip(request)
		if err != nil {
			labels := prometheus.Labels{"error": "err"}
			counter.With(labels).Inc()
		}

		return response, err
	})
}

// DecorateClientWithMetrics produces an http.RoundTripper from the configured Outbounder
// that is also decorated with appropriate metrics.
//
// RequestCounter, RequestDuration, DroppedMessages Inflight,
func DecorateClientWithMetrics(om OutboundMeasures, roundtripper http.RoundTripper) http.RoundTripper {
	return promhttp.RoundTripperFunc(InstrumentOutboundCounter(om.RequestCounter,
		InstrumentOutboundDuration(om.RequestDuration,
			InstrumentOutboundDroppedMessages(om.DroppedMessages,
				promhttp.InstrumentRoundTripperInFlight(om.InFlight, roundtripper)))))
}
