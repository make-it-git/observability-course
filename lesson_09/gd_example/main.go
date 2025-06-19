package main

import (
	"fmt"
	"math"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var cpuConsumingFeatureEnabled int32 = 1

var (
	requestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_latency_seconds",
		Help:    "Request latency in seconds.",
		Buckets: prometheus.ExponentialBuckets(0.1, 1.2, 10),
	})

	featureFlagEnabled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "feature_flag_enabled",
			Help: "Whether a feature flag is enabled (1) or disabled (0).",
		},
		[]string{"feature"},
	)
)

func init() {
	prometheus.MustRegister(requestLatency)
	prometheus.MustRegister(featureFlagEnabled)
}

func burnCPU(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if atomic.LoadInt32(&cpuConsumingFeatureEnabled) == 1 {
		x := 1.0
		for i := 0; i < 1_000_000; i++ {
			x = math.Sin(math.Cos(math.Tan(math.Atan(x))))
		}
		fmt.Println("Time consuming thing")
	} else {
		x := 1.0
		for i := 0; i < 1_000; i++ {
			x = math.Sin(math.Cos(math.Tan(math.Atan(x))))
		}
		fmt.Println("Simple calculations, degraded service quality")
	}

	duration := time.Since(start).Seconds()
	requestLatency.Observe(duration)

	fmt.Fprintln(w, "OK")
}

func degradeHandler(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt32(&cpuConsumingFeatureEnabled, 0)
	featureFlagEnabled.With(prometheus.Labels{"feature": "cpu_burning_feature"}).Set(0)
	fmt.Fprintln(w, "Degradation enabled")
}

func undegradeHandler(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt32(&cpuConsumingFeatureEnabled, 1)
	featureFlagEnabled.With(prometheus.Labels{"feature": "cpu_burning_feature"}).Set(1)
	fmt.Fprintln(w, "Degradation disabled")
}

func main() {
	featureFlagEnabled.With(prometheus.Labels{"feature": "cpu_burning_feature"}).Set(1)

	http.HandleFunc("/process", burnCPU)
	http.HandleFunc("/degrade", degradeHandler)
	http.HandleFunc("/undegrade", undegradeHandler)
	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
