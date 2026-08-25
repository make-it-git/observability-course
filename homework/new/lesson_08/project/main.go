package main

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const serviceName = "orders-service"

var (
	// Histogram metric (automatically produces _count, _sum, and _bucket series)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_server_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service_name", "path", "http_response_status_code"},
	)

	reqCounter uint64
)

func init() {
	prometheus.MustRegister(requestDuration)
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	count := atomic.AddUint64(&reqCounter, 1)

	// Demo: Каждый 3-й запрос зависает
	if count%3 == 0 && false {
		time.Sleep(1200 * time.Millisecond) // Каскадная задержка

		requestDuration.WithLabelValues(serviceName, "/orders", "500").Observe(time.Since(start).Seconds())

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "error", "message": "Upstream service timeout"}`))
		return
	}

	time.Sleep(50 * time.Millisecond)

	requestDuration.WithLabelValues(serviceName, "/orders", "200").Observe(time.Since(start).Seconds())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

func main() {
	http.HandleFunc("/orders", orderHandler)
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Server running on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
