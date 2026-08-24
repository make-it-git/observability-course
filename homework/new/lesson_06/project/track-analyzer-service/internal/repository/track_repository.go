package repository

import (
	"context"
	"encoding/json"
	"example/track-analyzer-service/internal/domain/models"
	"example/track-analyzer-service/internal/metrics"
	"example/track-analyzer-service/internal/service"
	"example/track-analyzer-service/internal/tracing"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel/codes"
)

type TrackRepository interface {
	SaveTrackAnalysis(ctx context.Context, analysis *models.TrackAnalysis) error
	GetRecentPoints(ctx context.Context, driverID string, count int) ([]models.GpsPoint, error)
	SavePoint(ctx context.Context, driverID string, pointData []byte) error
}

type redisTrackRepository struct {
	client         *redis.Client
	trackService   *service.TrackService
	featureService *service.FeatureService
}

func NewRedisTrackRepository(client *redis.Client, featureService *service.FeatureService) TrackRepository {
	return &redisTrackRepository{
		client:         client,
		trackService:   service.NewTrackService(),
		featureService: featureService,
	}
}

func (r *redisTrackRepository) SaveTrackAnalysis(ctx context.Context, analysis *models.TrackAnalysis) error {
	ctx, span := tracing.TrackAnalysisSpan(ctx, analysis.DriverID)
	defer span.End()

	key := fmt.Sprintf("track:analysis:%s", analysis.DriverID)
	data, err := json.Marshal(analysis)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal analysis")
		return fmt.Errorf("failed to marshal analysis: %w", err)
	}

	dummyValue := 0.0
	if r.featureService.GetBoolFeature(ctx, "slow-save-track-analysis", false) {
		span.AddEvent("slow-save-point")
		for i := 0; i < 1000_000; i++ {
			x := i * 100500
			dummyValue = float64(x) + rand.Float64()
		}
	}
	_ = dummyValue

	err = r.client.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save analysis to Redis")
		return err
	}

	span.SetStatus(codes.Ok, "analysis saved successfully")
	return nil
}

func (r *redisTrackRepository) GetRecentPoints(ctx context.Context, driverID string, count int) ([]models.GpsPoint, error) {
	ctx, span := tracing.GetPointsSpan(ctx, driverID, count)
	defer span.End()

	start := time.Now()
	defer func() {
		metrics.AnalysisLatency.WithLabelValues(driverID).Observe(time.Since(start).Seconds())
	}()

	key := fmt.Sprintf("track:points:%s", driverID)
	data, err := r.client.LRange(ctx, key, 0, int64(count-1)).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get points from Redis")
		return nil, fmt.Errorf("failed to get recent points: %w", err)
	}

	points := make([]models.GpsPoint, 0, len(data))
	for _, item := range data {
		var point models.GpsPoint
		if err := json.Unmarshal([]byte(item), &point); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal point")
			return nil, fmt.Errorf("failed to unmarshal point: %w", err)
		}
		points = append(points, point)
	}

	// If we have points, analyze them
	if len(points) > 0 {
		metrics.PointsInAnalysis.WithLabelValues(driverID).Observe(float64(len(points)))
		analysis := r.trackService.AnalyzeTrack(ctx, points)
		return analysis.Points, nil
	}

	span.SetStatus(codes.Ok, "points retrieved successfully")
	return points, nil
}

func (r *redisTrackRepository) SavePoint(ctx context.Context, driverID string, pointData []byte) error {
	ctx, span := tracing.SavePointSpan(ctx, driverID)
	defer span.End()

	start := time.Now()
	defer func() {
		metrics.AnalysisLatency.WithLabelValues(driverID).Observe(time.Since(start).Seconds())
	}()

	key := fmt.Sprintf("track:points:%s", driverID)

	// Save the point
	if err := r.client.LPush(ctx, key, pointData).Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to save point to Redis")
		return fmt.Errorf("failed to save point: %w", err)
	}

	// Increment processed points counter
	metrics.ProcessedPoints.WithLabelValues(driverID).Inc()

	// Get last 100 points for analysis
	data, err := r.client.LRange(ctx, key, 0, 99).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get points for analysis")
		return fmt.Errorf("failed to get points for analysis: %w", err)
	}

	points := make([]models.GpsPoint, 0, len(data))
	for _, item := range data {
		var point models.GpsPoint
		if err := json.Unmarshal([]byte(item), &point); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to unmarshal point for analysis")
			return fmt.Errorf("failed to unmarshal point for analysis: %w", err)
		}
		points = append(points, point)
	}

	// Analyze track if we have enough points
	if len(points) > 0 {
		metrics.PointsInAnalysis.WithLabelValues(driverID).Observe(float64(len(points)))
		analysis := r.trackService.AnalyzeTrack(ctx, points)
		if err := r.SaveTrackAnalysis(ctx, analysis); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to save track analysis")
			return fmt.Errorf("failed to save track analysis: %w", err)
		}
	}

	span.SetStatus(codes.Ok, "point saved and analyzed successfully")
	return nil
}
