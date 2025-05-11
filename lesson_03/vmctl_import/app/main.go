package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Определяем метрики
var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"path", "method", "status", "app_instance"}) // Добавлена метка instance

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "method", "status", "app_instance"}) // Добавлена метка instance
)

func PrometheusMiddleware(appInstance string) gin.HandlerFunc { // Принимаем appInstance
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
			"path":         path,
			"method":       c.Request.Method,
			"status":       string(strconv.Itoa(statusCode)[0]),
			"app_instance": appInstance, // Добавляем метку appInstance
		}).Inc()

		httpRequestDuration.With(prometheus.Labels{
			"path":         path,
			"method":       c.Request.Method,
			"status":       string(strconv.Itoa(statusCode)[0]),
			"app_instance": appInstance, // Добавляем метку appInstance
		}).Observe(duration.Seconds())
	}
}

func main() {
	router := gin.Default()

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	appInstance, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v, using 'unknown'", err)
	}
	appInstance = fmt.Sprintf("%s:%s", appInstance, appPort)

	router.Use(PrometheusMiddleware(appInstance))

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	appAddr := fmt.Sprintf(":%s", appPort)
	fmt.Printf("Starting application server on %s\n", appAddr)
	if err := router.Run(appAddr); err != nil {
		log.Fatalf("Failed to start application server: %v", err)
	}
}
