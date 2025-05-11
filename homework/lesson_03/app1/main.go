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
		Name:    "http_requests_latency_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.1, 0.5, 1, 2, 7, 12, 20, 30},
	}, []string{"method", "path", "status"})
)

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()
	r.Use(middleware)

	r.HandleFunc("/get-user/{userID}", getUserHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())

	fmt.Println("Service listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["userID"]
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Got user %s\n", userID)))
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		// Hint: need to replace r.URL.Path with parameterized Path to avoid cardinality explosion
		httpRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path, "status": statusCode},
		).Inc()

		httpRequestsLatency.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path, "status": statusCode}).
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
