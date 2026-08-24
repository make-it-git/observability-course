package main

import (
	"math"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Неожиданное изменение тренда
	linearMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "linear_metric",
		Help: "Metric that grows linearly but changes trend.",
	})

	// Отклонение от сезонного паттерна
	sinusoidalMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sinusoidal_metric",
		Help: "Metric that follows a sinusoidal pattern.",
	})

	// Обнаружение аномальных всплесков
	spikeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "spike_metric",
		Help: "Metric with occasional spikes.",
	})
)

var (
	// linearMetric
	currentValue atomic.Int64
	step         int64 = 10

	// sinusoidalMetric
	amplitude  = 10.0
	frequency  = 0.1
	timeOffset = 0.0

	// spikeMetric
	baseValue      = 5.0
	spikeChance    = 0.3
	spikeMagnitude = 25.0
)

func main() {
	rand.Seed(time.Now().UnixNano())

	go simulation("trend_change")
	go simulation("seasonal_deviation")
	go simulation("spike_detection")

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func simulation(simulationMode string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	trendChangeTime := time.Now().Add(300 * time.Second)   // Change the trend after 300 seconds
	patternChangeTime := time.Now().Add(300 * time.Second) // change the Pattern after 300 seconds

	for range ticker.C {
		switch simulationMode {
		case "trend_change":
			now := time.Now()
			if now.After(trendChangeTime) {
				// Change the trend:  Slow down the growth
				step = 1 // Reduced step size
			}

			newValue := currentValue.Add(step)
			linearMetric.Set(float64(newValue))

		case "seasonal_deviation":
			now := time.Now()
			if now.After(patternChangeTime) {
				// Change the pattern: Reduce the amplitude
				amplitude = 5.0
			}

			timeOffset++
			value := amplitude * math.Sin(2*math.Pi*frequency*timeOffset)
			sinusoidalMetric.Set(value)

		case "spike_detection":
			value := baseValue

			// Add a spike with a certain probability
			if rand.Float64() < spikeChance {
				value += spikeMagnitude
			}

			spikeMetric.Set(value)
		}
	}
}
