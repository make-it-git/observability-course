package service

import (
	"bytes"
	"context"
	"encoding/json"
	"example/driver-location-service/internal/domain/models"
	"fmt"
	"go.opentelemetry.io/otel/codes"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type DriverService interface {
	UpdateLocation(ctx context.Context, driverID string, location models.Location) error
	FindNearby(ctx context.Context, lat, lon float64, radius float64) ([]models.Driver, error)
	RemoveDriver(ctx context.Context, driverID string) error
}

type driverService struct {
	redis       *redis.Client
	trackingURL string
	logger      *slog.Logger
}

func NewDriverService(redis *redis.Client, trackingURL string, logger *slog.Logger) DriverService {
	return &driverService{
		redis:       redis,
		trackingURL: trackingURL,
		logger:      logger,
	}
}

func (s *driverService) UpdateLocation(ctx context.Context, driverID string, location models.Location) error {
	driver := models.Driver{
		ID:       driverID,
		Location: location,
		Status:   "active",
	}

	// Store driver data
	key := fmt.Sprintf("driver:%s", driverID)
	data, err := json.Marshal(driver)
	if err != nil {
		return fmt.Errorf("failed to marshal driver: %w", err)
	}

	// Update geospatial index
	geoKey := "driver:locations"
	if err := s.redis.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: location.Longitude,
		Latitude:  location.Latitude,
	}).Err(); err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}

	// Send to track analyzer service
	point := models.GpsPoint{
		DriverID:  driverID,
		Location:  location,
		Timestamp: time.Now().Unix(),
	}

	go func() {
		spanCtx := trace.SpanContextFromContext(ctx)
		logger := s.logger.With(
			slog.String("traceID", spanCtx.TraceID().String()),
			slog.String("driverID", driverID),
		)

		if err := s.sendToTrackAnalyzer(ctx, point); err != nil {
			logger.Error("failed to send point to track analyzer",
				"error", err,
				"latitude", location.Latitude,
				"longitude", location.Longitude,
			)
		}
	}()

	return s.redis.Set(ctx, key, data, 0).Err()
}

func (s *driverService) sendToTrackAnalyzer(ctx context.Context, point models.GpsPoint) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// Create a new span for the HTTP request
	ctx, span := otel.Tracer("driver-service").Start(ctx, "sendToTrackAnalyzer", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	logger := s.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
		slog.String("driverID", point.DriverID),
	)

	data, err := json.Marshal(point)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal point")
		logger.Error("failed to marshal point",
			"error", err,
			"latitude", point.Location.Latitude,
			"longitude", point.Location.Longitude,
		)
		return fmt.Errorf("failed to marshal point: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/tracks/%s/points", s.trackingURL, point.DriverID),
		bytes.NewBuffer(data))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request")
		logger.Error("failed to create request",
			"error", err,
			"url", s.trackingURL,
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Inject trace context into HTTP headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send request")
		logger.Error("failed to send request",
			"error", err,
			"url", req.URL.String(),
		)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, "unexpected status code from track analyzer")
		logger.Error("unexpected status code from track analyzer",
			"statusCode", resp.StatusCode,
			"url", req.URL.String(),
		)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	span.SetStatus(codes.Ok, "sent location")
	return nil
}

func (s *driverService) FindNearby(ctx context.Context, lat, lon float64, radius float64) ([]models.Driver, error) {
	spanCtx := trace.SpanContextFromContext(ctx)
	logger := s.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
		slog.Float64("latitude", lat),
		slog.Float64("longitude", lon),
		slog.Float64("radius", radius),
	)

	geoKey := "driver:locations"

	// Find drivers within radius (in meters)
	locations, err := s.redis.GeoRadius(ctx, geoKey, lon, lat, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      "m",
		WithCoord: true,
		WithDist:  true,
		Count:     50,
	}).Result()

	if err != nil {
		logger.Error("failed to query locations", "error", err)
		return nil, fmt.Errorf("failed to query locations: %w", err)
	}

	drivers := make([]models.Driver, 0, len(locations))
	for _, loc := range locations {
		key := fmt.Sprintf("driver:%s", loc.Name)
		data, err := s.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue // Skip if driver data not found
		}

		var driver models.Driver
		if err := json.Unmarshal(data, &driver); err != nil {
			continue
		}
		drivers = append(drivers, driver)
	}

	return drivers, nil
}

func (s *driverService) RemoveDriver(ctx context.Context, driverID string) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	logger := s.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
		slog.String("driverID", driverID),
	)

	geoKey := "driver:locations"
	key := fmt.Sprintf("driver:%s", driverID)

	if err := s.redis.ZRem(ctx, geoKey, driverID).Err(); err != nil {
		logger.Error("failed to remove from geo index", "error", err)
		return fmt.Errorf("failed to remove from geo index: %w", err)
	}

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		logger.Error("failed to remove driver data", "error", err)
		return fmt.Errorf("failed to remove driver data: %w", err)
	}

	return nil
}
