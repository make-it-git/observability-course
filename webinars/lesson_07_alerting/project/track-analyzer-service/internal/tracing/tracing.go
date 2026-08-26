package tracing

import (
	"context"
	"example/track-analyzer-service/internal/domain/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("track-analyzer-service")

// StartSpan starts a new span with the given name and attributes
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// TrackAnalysisSpan creates a span for track analysis operations
func TrackAnalysisSpan(ctx context.Context, driverID string) (context.Context, trace.Span) {
	return StartSpan(ctx, "track.analysis",
		attribute.String("driver_id", driverID),
	)
}

// SavePointSpan creates a span for saving GPS points
func SavePointSpan(ctx context.Context, driverID string) (context.Context, trace.Span) {
	return StartSpan(ctx, "track.save_point",
		attribute.String("driver_id", driverID),
	)
}

// GetPointsSpan creates a span for retrieving GPS points
func GetPointsSpan(ctx context.Context, driverID string, count int) (context.Context, trace.Span) {
	return StartSpan(ctx, "track.get_points",
		attribute.String("driver_id", driverID),
		attribute.Int("count", count),
	)
}

// AnalyzeTrackSpan creates a span for track analysis
func AnalyzeTrackSpan(ctx context.Context, points []models.GpsPoint) (context.Context, trace.Span) {
	if len(points) == 0 {
		return StartSpan(ctx, "track.analyze",
			attribute.Int("points_count", 0),
		)
	}
	return StartSpan(ctx, "track.analyze",
		attribute.String("driver_id", points[0].DriverID),
		attribute.Int("points_count", len(points)),
	)
}
