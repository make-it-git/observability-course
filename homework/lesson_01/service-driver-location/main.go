// driver_location.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	driverLocationHTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	http500ErrorsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_500_error_total",
		Help: "Total number of requests with 500 status code.",
	}, []string{"path"})

	driverLocationHTTPRequestsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_requests_latency_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.1, 0.5, 1, 2, 7, 12, 20, 30},
	}, []string{"method", "path"})

	// Incorrect Metric: This metric is INCORRECTLY measuring the total number of drivers and exposing it as a gauge.
	// A gauge should measure the *current* number, but we're just incrementing, so it will just keep growing forever.
	// A more useful metric would be a gauge that's updated periodically with the current count of available drivers.
	totalDriversGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_drivers",
		Help: "Current number of drivers",
	})

	// A better metric will be a current number of available drivers
	availableDriversGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "available_drivers",
		Help: "Current number of available drivers",
	})
)

var availableDrivers = []string{"driver1", "driver2", "driver3", "driver4", "driver5"}

// Atomic counter for total number of drivers (used for the INCORRECT metric)
var totalDrivers int64 = int64(len(availableDrivers))

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()

	r.Use(middleware)

	r.HandleFunc("/available-drivers", availableDriversHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())

	// Start a goroutine to simulate driver availability updates
	go updateDriverAvailability()

	fmt.Println("Driver Location service listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func availableDriversHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate random slow down, approx 10% chance
	// Imagine that we ignore error returned from the slow database and return empty list here
	if rand.Intn(10) == 0 {
		log.Println("Querying database")
		time.Sleep(time.Second * 10) // Slow response, we face timeout from database
		// add metric for 500 errors if db returns timeout
		http500ErrorsCounter.WithLabelValues("/avalible-drives").Add(1)
		// fix to throwing error Database timeout
		http.Error(w, "Database timeout", http.StatusInternalServerError)
		return
	}

	available := getAvailableDrivers() // Get the current snapshot of available drivers
	json.NewEncoder(w).Encode(available)
	log.Printf("Returning available drivers: %v\n", available)

	// Update the 'availableDriversGauge' with the current number of available drivers
	availableDriversGauge.Set(float64(len(available)))
}

func getAvailableDrivers() []string {
	available := make([]string, len(availableDrivers))
	copy(available, availableDrivers)
	return available
}

func updateDriverAvailability() {
	for {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second) // Simulate random intervals
		// Simulate driver joining or leaving
		if rand.Intn(2) == 0 { // 50% chance of a driver becoming available
			newDriver := fmt.Sprintf("driver%d", rand.Intn(100)+6) // Create a new driver ID
			availableDrivers = append(availableDrivers, newDriver)
			log.Printf("Driver joined: %s\n", newDriver)
			atomic.AddInt64(&totalDrivers, 1) // Update the (INCORRECT) total drivers count
		} else { // 50% chance of a driver becoming unavailable
			if len(availableDrivers) > 0 {
				indexToRemove := rand.Intn(len(availableDrivers))
				driverToRemove := availableDrivers[indexToRemove]
				availableDrivers = append(availableDrivers[:indexToRemove], availableDrivers[indexToRemove+1:]...)
				log.Printf("Driver left: %s\n", driverToRemove)
				atomic.AddInt64(&totalDrivers, -1)
			}
		}
		// Update the 'totalDriversGauge' with the current number of total drivers after changes
		totalDriversGauge.Set(float64(totalDrivers))
		// Update the 'availableDriversGauge' with the current number of available drivers after changes
		availableDriversGauge.Set(float64(len(availableDrivers)))
	}
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		driverLocationHTTPRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path, "status": statusCode},
		).Inc()

		driverLocationHTTPRequestsLatency.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path},
		).Observe(duration.Seconds())
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
