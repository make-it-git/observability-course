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

	ProcessedPoints = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "processed_gps_points_total",
			Help: "Total number of processed GPS points",
		},
		[]string{"driver_id"},
	)

	AnalysisLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "track_analysis_duration_seconds",
			Help:    "Time spent analyzing GPS tracks",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"driver_id"},
	)

	GpsAccuracy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gps_accuracy_meters",
			Help: "Estimated GPS accuracy in meters",
		},
		[]string{"driver_id"},
	)

	PointsInAnalysis = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "points_in_analysis",
			Help:       "Number of points used in track analysis",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"driver_id"},
	)
)
