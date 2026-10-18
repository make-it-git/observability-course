package middleware

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strconv"
	"time"

	"example/track-analyzer-service/internal/metrics"

	"github.com/go-chi/chi/v5"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Status() int {
	return w.status
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		ww := NewResponseWriter(w)
		next.ServeHTTP(ww, r)

		routePattern := chi.RouteContext(ctx).RoutePattern()
		if routePattern == "" {
			routePattern = "undefined"
		}

		spanCtx := trace.SpanContextFromContext(ctx)
		traceID := spanCtx.TraceID().String()
		traceLabels := prometheus.Labels{"trace_id": traceID}

		metrics.HttpRequestsTotal.WithLabelValues(
			routePattern,
			r.Method,
			strconv.Itoa(ww.Status()),
		).(prometheus.ExemplarAdder).AddWithExemplar(1, traceLabels)

		metrics.HttpRequestDuration.WithLabelValues(
			routePattern,
			r.Method,
		).(prometheus.ExemplarObserver).ObserveWithExemplar(time.Since(start).Seconds(), traceLabels)

		metrics.HttpRequestDurationSummary.WithLabelValues(
			routePattern,
			r.Method,
		).Observe(time.Since(start).Seconds())
	})
}
