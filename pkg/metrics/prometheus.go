package metrics

import (
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	RequestCount   *kitprometheus.Counter
	RequestLatency *kitprometheus.Histogram
}

func NewGPSMetrics() *Metrics {
	return &Metrics{
		RequestCount: kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "gps",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"transport", "path", "rpc", "status"}),

		RequestLatency: kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
			Namespace: "gps",
			Name:      "request_latency_seconds",
			Help:      "Time spent processing requests.",
		}, []string{"transport", "path", "rpc", "status"}),
	}
}
