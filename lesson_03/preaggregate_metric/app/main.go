package main

import (
	"container/ring"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	windowSize = 15 // Size of the sliding window in seconds
)

var (
	instantaneousValueGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "instantaneous_value",
		Help: "Instantaneous value that can spike",
	})

	slidingWindowMaxValueGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sliding_window_max_value",
		Help: "Maximum value over the sliding window",
	})

	slidingWindowMedianValueGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sliding_window_median_value",
		Help: "Median value over the sliding window",
	})

	valueRing = ring.New(windowSize) // Ring buffer
	mutex     = &sync.Mutex{}
)

func init() {
	prometheus.MustRegister(instantaneousValueGauge, slidingWindowMaxValueGauge, slidingWindowMedianValueGauge)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	go simulateSpikyValue()
	go calculateSlidingWindowStats()

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func simulateSpikyValue() {
	value := 5.0
	i := 0
	for {
		i++
		value = float64(i)
		if i%12 == 0 {
			value = 100.0
		} else {
			value = 5.0
		}

		instantaneousValueGauge.Set(value)

		mutex.Lock()
		valueRing.Value = value
		valueRing = valueRing.Next()
		mutex.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func calculateSlidingWindowStats() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		values := make([]float64, 0, windowSize)
		maxValue := 0.0

		mutex.Lock()
		valueRing.Do(func(p interface{}) {
			if p != nil {
				value := p.(float64)
				values = append(values, value)
				if value > maxValue {
					maxValue = value
				}
			}
		})
		mutex.Unlock()

		fmt.Printf("values=%v\n", values)
		sort.Float64s(values)
		median := calculateMedian(values)

		slidingWindowMaxValueGauge.Set(maxValue)
		slidingWindowMedianValueGauge.Set(median)
		fmt.Printf("maxValue=%f, median=%f\n", maxValue, median)
	}
}

func calculateMedian(values []float64) float64 {
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}
