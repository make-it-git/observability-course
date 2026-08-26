package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)
	HttpRequestDurationSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_duration_seconds_summary",
			Help: "HTTP request duration in seconds (summary)",
			Objectives: map[float64]float64{
				0.5:  0.05,  // 50th percentile, +/- 5%
				0.75: 0.01,  // 75th percentile, +/- 1%
				0.9:  0.01,  // 90th percentile, +/- 1%
				0.95: 0.005, // 95th percentile, +/- 0.5%
				0.99: 0.001, // 99th percentile, +/- 0.1%
			},
		},
		[]string{"path", "method"},
	)

	LocationUpdates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "driver_location_updates_total",
			Help: "Total number of driver location updates",
		},
		[]string{"driver_id"},
	)
)
