package middleware

import (
	"go.opentelemetry.io/otel/trace"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// TracingMiddleware extracts trace context from incoming HTTP requests
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from request headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "undefined"
		}

		// Start a new span
		ctx, span := otel.Tracer("http").Start(ctx, "http_request", trace.WithSpanKind(trace.SpanKindServer))
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", routePattern),
		)
		defer span.End()

		// Create a new request with the span context
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
