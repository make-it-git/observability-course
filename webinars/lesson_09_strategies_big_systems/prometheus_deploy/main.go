package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
)

var (
	requestCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "my_go_app_requests_total",
		Help: "Total number of requests to the app.",
	})
	errorCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "my_go_app_errors_total",
		Help: "Total number of errors in the app.",
	})
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestCount.Inc()
		if rand.Float64() < 0.1 { // Simulate a 10% error rate
			errorCount.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Error!")
			return
		}
		fmt.Fprintln(w, "Hello, World!")
	})

	http.Handle("/metrics", promhttp.Handler())

	fmt.Println("Starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
