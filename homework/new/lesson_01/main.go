package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "handler", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(next http.HandlerFunc, handlerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(rw, r)

		httpRequestsTotal.WithLabelValues(r.Method, handlerName, "200").Inc()
	}
}

func processOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Имитация нестабильной работы внешнего платежного шлюза
	n := rand.Intn(100)
	if n < 30 {
		time.Sleep(time.Duration(50+rand.Intn(50)) * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"error","message":"payment_gateway_timeout"}`))
		return
	}

	time.Sleep(time.Duration(10+rand.Intn(20)) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","order_id":"ORD-9921"}`))
}

func main() {
	http.HandleFunc("/api/v1/orders", metricsMiddleware(processOrderHandler, "process_order"))
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Order Processor running on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
