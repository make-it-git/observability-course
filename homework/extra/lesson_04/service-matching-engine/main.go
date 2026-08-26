package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	matchingEngineHTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	matchingEngineHTTPRequestsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_requests_latency_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.1, 0.5, 1, 2, 7, 12, 20, 30},
	}, []string{"method", "path"})
)

var driverLocationURL = os.Getenv("DRIVER_LOCATION_URL")

func main() {
	if driverLocationURL == "" {
		driverLocationURL = "http://driver-location:8080"
	}
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()

	r.Use(middleware)

	r.HandleFunc("/find-driver", findDriverHandler).Methods("POST")
	r.Handle("/metrics", promhttp.Handler())

	fmt.Println("Matching Engine service listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func findDriverHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get(driverLocationURL + "/available-drivers")
	if err != nil {
		http.Error(w, "Error connecting to Driver Location", http.StatusInternalServerError)
		log.Printf("Error connecting to Driver Location: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Driver Location returned an error", resp.StatusCode)
		log.Printf("Driver Location returned error status: %d\n", resp.StatusCode)
		return
	}

	var drivers []string
	err = json.NewDecoder(resp.Body).Decode(&drivers)
	if err != nil {
		http.Error(w, "Error decoding Driver Location response", http.StatusInternalServerError)
		log.Printf("Error decoding Driver Location response: %v\n", err)
		return
	}

	if len(drivers) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "No drivers available"})
		log.Println("No drivers available")
		return
	}

	// Randomly select a driver
	randomIndex := rand.Intn(len(drivers))
	selectedDriver := drivers[randomIndex]

	json.NewEncoder(w).Encode(map[string]string{"driver": selectedDriver})
	log.Printf("Matched driver: %s\n", selectedDriver)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		matchingEngineHTTPRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path, "status": statusCode},
		).Inc()

		matchingEngineHTTPRequestsLatency.With(
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
