package handlers

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"product-catalog/internal/metrics"
	"strconv"
	"time"
)

func HttpMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		statusCode := strconv.Itoa(ww.statusCode)

		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate() // Get the route pattern

		metrics.HTTPRequestsTotal.With(
			prometheus.Labels{"method": r.Method, "path": path, "status": statusCode},
		).Inc()

		metrics.HTTPRequestsLatency.With(
			prometheus.Labels{"method": r.Method, "path": path, "status": statusCode},
		).Observe(duration.Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (ww *responseWriter) WriteHeader(statusCode int) {
	ww.statusCode = statusCode
	ww.ResponseWriter.WriteHeader(statusCode)
}
