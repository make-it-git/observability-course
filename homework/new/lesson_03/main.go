package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response latency for HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(rw, r)

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(path, strconv.Itoa(rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(path, "200").Observe(duration)
	}
}

func handleCart(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","cart_id":123}`))
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	// Сценарий: 5% запросов на checkout падает с таймаутом/ошибкой 503 и задержкой 1.5s
	if rand.Float32() < 0.05 {
		time.Sleep(1500 * time.Millisecond)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"payment gateway timeout"}`))
		return
	}

	time.Sleep(time.Duration(50+rand.Intn(50)) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","order_id":999}`))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/api/v1/cart", metricsMiddleware("/api/v1/cart", handleCart))
	http.HandleFunc("/api/v1/checkout", metricsMiddleware("/api/v1/checkout", handleCheckout))
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
