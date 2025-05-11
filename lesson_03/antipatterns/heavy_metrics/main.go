package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"time"
)

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

// This fuction runs when Prometheus scrapes the exporter
func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	time.Sleep(time.Second * 3)
	// Here we set new value for the metric
	ch <- prometheus.MustNewConstMetric(collector.heavyMetric, prometheus.GaugeValue, rand.Float64())
}

func main() {
	collector := newCollector()
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
