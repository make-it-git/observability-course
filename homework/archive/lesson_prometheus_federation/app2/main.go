package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	httpRequestsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_requests_latency_seconds",
		Help: "HTTP request latency in seconds.",
		// Hint: too many buckets
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 500),
	}, []string{"method", "path", "status"})
)

func main() {
	rand.Seed(time.Now().UnixNano())

	collector := newCollector()
	prometheus.MustRegister(collector)

	r := mux.NewRouter()
	r.Use(middleware)

	r.HandleFunc("/send-message/{userID}", sendMessageHandler).Methods("POST")
	r.Handle("/metrics", promhttp.Handler())

	fmt.Println("Service listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userID"]
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Message sent to %s\n", userID)))
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate() // Get the route pattern
		httpRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": path, "status": statusCode},
		).Inc()

		httpRequestsLatency.With(
			prometheus.Labels{"method": r.Method, "path": path, "status": statusCode}).
			Observe(duration.Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (ww *responseWriter) WriteHeader(statusCode int) {
	ww.statusCode = statusCode
	ww.ResponseWriter.WriteHeader(statusCode)
}

type Collector struct {
	heavyMetric *prometheus.Desc
}

func newCollector() *Collector {
	return &Collector{
		heavyMetric: prometheus.NewDesc("my_heavy_metric",
			"Takes long to calculate",
			nil, nil,
		),
	}
}

func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.heavyMetric
}

func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Println("Running collection")
	ch <- prometheus.MustNewConstMetric(collector.heavyMetric, prometheus.GaugeValue, calculateHeavyMetric())
}

func calculateHeavyMetric() float64 {
	time.Sleep(time.Second * time.Duration(rand.Intn(5)+1))
	return rand.Float64()
}
