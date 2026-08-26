package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics
var (
	aggregatedProcessGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "process_value_aggregated",
		Help: "Aggregated gauge value of the process (e.g., average over the scrape interval).",
	}, []string{"process_name"})

	scrapeCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "prometheus_scrapes_total",
		Help: "Total number of Prometheus scrapes.",
	})
)

const numProcesses = 5
const scrapeInterval = 10 * time.Second

// ProcessData stores per-second values.
type ProcessData struct {
	values []float64
	mu     sync.Mutex
}

// perProcessData stores data for each process.
var perProcessData = make(map[string]*ProcessData)

// emulateProcess function now stores per-second values.
func emulateProcess(processName string, stop chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	data, ok := perProcessData[processName]
	if !ok {
		perProcessData[processName] = &ProcessData{values: make([]float64, 0, int64(scrapeInterval.Seconds()))} // Pre-allocate space
		data = perProcessData[processName]
	}

	for {
		select {
		case <-ticker.C:
			value := rand.Float64() * 100
			if processName == "process-4" {
				value = rand.Float64() * 10 // Simulate process difference from other processes
			}
			data.mu.Lock()
			data.values = append(data.values, value)
			data.mu.Unlock()

		case <-stop:
			log.Printf("Process %s stopped\n", processName)
			return
		}
	}
}

// aggregateAndExport aggregates the per-second data and exports to Prometheus.
func aggregateAndExport() {
	for processName, data := range perProcessData {
		data.mu.Lock()
		sum := 0.0
		for _, v := range data.values {
			sum += v
		}
		var average float64
		if len(data.values) > 0 {
			average = sum / float64(len(data.values))
		} else {
			average = 0
		}
		data.values = data.values[:0] // Clear the stored values
		data.mu.Unlock()

		aggregatedProcessGauge.With(prometheus.Labels{"process_name": processName}).Set(average)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	processStops := make([]chan bool, numProcesses)
	var wg sync.WaitGroup

	// Start emulated processes
	for i := 0; i < numProcesses; i++ {
		processName := fmt.Sprintf("process-%d", i)
		processStops[i] = make(chan bool)
		wg.Add(1)

		go func(name string, stop chan bool) {
			defer wg.Done()
			emulateProcess(name, stop)
		}(processName, processStops[i])
	}

	// Prometheus endpoint
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Aggregate and Export to Prometheus on scrape interval
	scrapeTicker := time.NewTicker(scrapeInterval)
	defer scrapeTicker.Stop()

	scrapeDuration := 5 * time.Minute
	endTime := time.Now().Add(scrapeDuration)

	for time.Now().Before(endTime) {
		select {
		case <-scrapeTicker.C:
			log.Println("Simulating Prometheus scrape...")
			scrapeCounter.Inc()
			aggregateAndExport() // Aggregate and export to Prometheus
		}
	}

	// Stop emulated processes
	for _, stop := range processStops {
		close(stop)
	}

	wg.Wait()
	log.Println("Done.")
}
