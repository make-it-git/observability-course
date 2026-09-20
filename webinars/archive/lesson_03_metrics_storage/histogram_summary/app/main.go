package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestDurationHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_duration_seconds_histogram",
		Help:    "Histogram of request durations.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 10),
	})
	requestDurationHistogramExample2 = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_duration_seconds_histogram_example",
		Help:    "Histogram of request durations.",
		Buckets: prometheus.ExponentialBuckets(0.005, 2, 10),
	})

	requestDurationSummary = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "request_duration_seconds_summary",
		Help:       "Summary of request durations.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
)

func main() {
	prometheus.MustRegister(requestDurationHistogram)
	prometheus.MustRegister(requestDurationSummary)
	prometheus.MustRegister(requestDurationHistogramExample2)

	go func() {
		for {
			duration := rand.Float64() * 0.2 // random durations up to 200ms
			time.Sleep(time.Duration(duration * float64(time.Second)))

			requestDurationHistogram.Observe(duration)
			requestDurationSummary.Observe(duration)
			requestDurationHistogramExample2.Observe(0.1)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)

}
