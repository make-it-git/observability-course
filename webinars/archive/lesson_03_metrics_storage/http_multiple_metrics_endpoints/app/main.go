package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Метрика для общего endpoint `/metrics`
	httpRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	})

	// Метрика для частого endpoint `/metrics_frequent`
	activeUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_users",
		Help: "Number of active users.",
	})

	// Еще одна метрика для частого endpoint `/metrics_frequent`
	queueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "queue_length",
		Help: "Current queue length.",
	})
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf(r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Регистрируем отдельные registry для каждого endpoint
	commonRegistry := prometheus.NewRegistry()
	frequentRegistry := prometheus.NewRegistry()

	// Регистрируем метрики в соответствующих registry
	commonRegistry.MustRegister(httpRequestsTotal)
	frequentRegistry.MustRegister(activeUsers, queueLength)

	// Создаем обработчики для каждого endpoint, используя разные registry
	commonHandler := promhttp.HandlerFor(commonRegistry, promhttp.HandlerOpts{})
	frequentHandler := promhttp.HandlerFor(frequentRegistry, promhttp.HandlerOpts{})

	// Регистрируем обработчики на нужных путях
	http.Handle("/metrics", metricsMiddleware(commonHandler))
	http.Handle("/metrics_frequent", metricsMiddleware(frequentHandler))

	// Пример генерации данных для метрик
	go func() {
		for {
			httpRequestsTotal.Inc()
			time.Sleep(10 * time.Second) // Реже
		}
	}()

	go func() {
		for {
			activeUsers.Set(float64(time.Now().Second())) // Просто пример
			queueLength.Set(float64(time.Now().Minute())) // Просто пример
			time.Sleep(1 * time.Second)                   // Чаще
		}
	}()

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
