package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Определяем метрики
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"path", "method", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets, // Можно настроить buckets
	}, []string{"path", "method", "status"})
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		statusCode := c.Writer.Status()
		duration := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequestsTotal.With(prometheus.Labels{
			"path":   path,
			"method": c.Request.Method,
			"status": string(strconv.Itoa(statusCode)[0]), //Первая цифра status code (1xx,2xx,3xx...)
			// Но для кодов 4xx может понадобиться большая гранулярность.
		}).Inc()

		httpRequestDuration.With(prometheus.Labels{
			"path":   path,
			"method": c.Request.Method,
			"status": string(strconv.Itoa(statusCode)[0]), //Первая цифра status code (1xx,2xx,3xx...)
		}).Observe(duration.Seconds())
	}
}

func main() {
	router := gin.Default()

	router.Use(PrometheusMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	router.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.String(http.StatusOK, "User ID: "+id)
	})

	metricsPortStr := os.Getenv("METRICS_PORT")
	if metricsPortStr == "" {
		metricsPortStr = "10000"
	}

	metricsPort, err := strconv.Atoi(metricsPortStr)
	if err != nil {
		log.Fatalf("Invalid METRICS_PORT: %v", err)
	}

	metricsRouter := gin.New()
	metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Запускаем HTTP-сервер для метрик в отдельной горутине, не блокируя поток выполнения
	go func() {
		metricsAddr := fmt.Sprintf(":%d", metricsPort)
		fmt.Printf("Starting metrics server on %s\n", metricsAddr)
		if err := metricsRouter.Run(metricsAddr); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Запускаем основной HTTP-сервер
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	appAddr := fmt.Sprintf(":%s", appPort)
	fmt.Printf("Starting application server on %s\n", appAddr)
	if err := router.Run(appAddr); err != nil {
		log.Fatalf("Failed to start application server: %v", err)
	}
}
