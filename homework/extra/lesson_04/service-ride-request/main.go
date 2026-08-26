package main

import (
	"bytes"
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
	rideRequestHTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	rideRequestHTTPRequestsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_requests_latency_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: []float64{0.1, 0.5, 1, 2, 7, 12, 20, 30},
	}, []string{"method", "path"})
)

var matchingEngineURL = os.Getenv("MATCHING_ENGINE_URL")

func main() {
	if matchingEngineURL == "" {
		matchingEngineURL = "http://service-matching-engine:8080"
	}
	rand.Seed(time.Now().UnixNano())
	r := mux.NewRouter()

	// Middleware to handle metrics and logging
	r.Use(middleware)

	r.HandleFunc("/request-ride", requestRideHandler).Methods("POST")
	r.Handle("/metrics", promhttp.Handler())

	fmt.Println("Ride Request service listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func requestRideHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Decode the JSON body
	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Error decoding JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}
	defer r.Body.Close()

	// Extract pickup and dropoff locations from the decoded JSON
	pickup, okPickup := requestBody["pickup"]
	dropoff, okDropoff := requestBody["dropoff"]

	if !okPickup || !okDropoff {
		http.Error(w, "Missing pickup or dropoff parameters in JSON body", http.StatusBadRequest)
		log.Println("Missing pickup or dropoff parameters in JSON body")
		return
	}

	payload := map[string]string{"pickup": pickup, "dropoff": dropoff}
	payloadBytes, err := json.Marshal(payload)
	resp, err := http.Post(matchingEngineURL+"/find-driver", "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Error connecting to Matching Engine", http.StatusInternalServerError)
		log.Println("Error connecting to Matching Engine:", err) // Only log the error itself. Better to include more context
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			http.Error(w, "No drivers found", resp.StatusCode)
			return
		}
		http.Error(w, "Matching Engine returned an error", resp.StatusCode)
		// Incomplete logging - not including the error details, which is bad
		log.Printf("Matching Engine returned error status: %d\n", resp.StatusCode)
		return
	}

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error decoding Matching Engine response", http.StatusInternalServerError)
		log.Println("Error decoding Matching Engine response:", err)
		return
	}

	end := time.Since(start)
	if end.Seconds() < 1.0 {
		// Simulate real response time
		time.Sleep(time.Millisecond * 200)
	}
	json.NewEncoder(w).Encode(result)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		rideRequestHTTPRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": r.URL.Path, "status": statusCode},
		).Inc()

		rideRequestHTTPRequestsLatency.With(
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
