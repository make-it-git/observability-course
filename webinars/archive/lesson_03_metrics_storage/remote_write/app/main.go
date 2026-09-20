package main

import (
	"fmt"
	"github.com/marpaia/graphite-golang"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func sendToGraphite() {
	graphiteHost := os.Getenv("GRAPHITE_HOST")
	graphitePort, _ := strconv.ParseInt(os.Getenv("GRAPHITE_PORT"), 10, 64)
	graphite, err := graphite.NewGraphite(graphiteHost, int(graphitePort))
	if err != nil {
		log.Println("Error connecting to Graphite:", err)
		return
	}
	env := os.Getenv("ENV")
	defer graphite.Disconnect()

	for _, httpCode := range []int{200, 201, 202, 300, 400, 404, 500, 502} {
		count := 1
		if httpCode == 200 {
			count = 10
		}
		err = graphite.SimpleSend(fmt.Sprintf("env.%s.service_name.http.requests.%d.count", env, httpCode), fmt.Sprintf("%d", count))
		if err != nil {
			log.Println("Error sending metric to Graphite:", err)
		}
	}

	log.Println("graphite send completed")
}

func main() {
	numMetrics := 100_000

	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "generated_metric",
		Help: "Generated metric",
	}, []string{"my_label"})
	prometheus.MustRegister(metric)

	go func() {
		for {
			i := 0
			for i < numMetrics {
				metric.WithLabelValues(fmt.Sprintf("%d", i)).Set(float64(time.Now().UnixNano()))
				i++
			}
			sendToGraphite()
			time.Sleep(time.Second)
		}
	}()

	router := gin.Default()
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	addr := fmt.Sprintf(":8080")
	fmt.Printf("Starting metrics server on %s\n", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}
