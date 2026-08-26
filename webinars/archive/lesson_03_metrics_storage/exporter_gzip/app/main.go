package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	myCustomMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "my_custom_metric",
		Help: "A custom metric for example.",
	})
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		for {
			randomValue := rand.Float64() * 100 // Generate a random value between 0 and 100
			myCustomMetric.Set(randomValue)
			time.Sleep(time.Second)
			fmt.Printf("Metric updated to: %f\n", randomValue)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// DisableCompression: true,
		},
	))

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
