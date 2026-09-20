package service

import (
	"context"
	"example/track-analyzer-service/internal/domain/models"
	"example/track-analyzer-service/internal/metrics"
	"example/track-analyzer-service/internal/tracing"
	"math"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	maxReasonableSpeed = 200.0  // km/h
	earthRadius        = 6371.0 // km
)

type TrackService struct{}

func NewTrackService() *TrackService {
	return &TrackService{}
}

func (s *TrackService) AnalyzeTrack(ctx context.Context, points []models.GpsPoint) *models.TrackAnalysis {
	_, span := tracing.AnalyzeTrackSpan(ctx, points)
	defer span.End()

	start := time.Now()
	driverID := points[0].DriverID

	if len(points) < 2 {
		metrics.PointsInAnalysis.WithLabelValues(driverID).Observe(float64(len(points)))
		span.SetStatus(codes.Ok, "insufficient points for analysis")
		return &models.TrackAnalysis{
			DriverID:     driverID,
			Points:       points,
			AverageSpeed: 0,
			MaxSpeed:     0,
			Confidence:   1.0,
			Timestamp:    time.Now().Unix(),
		}
	}

	var totalSpeed float64
	var maxSpeed float64
	var anomalyCount int
	analyzedPoints := make([]models.GpsPoint, len(points))
	copy(analyzedPoints, points)

	// Analyze each point in relation to its previous point
	for i := 1; i < len(points); i++ {
		prevPoint := points[i-1]
		currentPoint := points[i]

		timeDiff := float64(prevPoint.Timestamp - currentPoint.Timestamp)
		if timeDiff == 0 {
			continue
		}

		distance := s.calculateDistance(prevPoint.Location, currentPoint.Location)
		speed := (distance * 3600) / timeDiff // Convert to km/h

		analysis := &models.PointAnalysis{
			CalculatedSpeed: speed,
			Confidence:      1.0,
			IsAnomaly:       false,
		}

		// Check for anomalies (unrealistic speeds)
		if speed > maxReasonableSpeed {
			analysis.IsAnomaly = true
			analysis.Confidence = maxReasonableSpeed / speed
			anomalyCount++

			// Calculate estimated location based on maximum reasonable speed
			maxDistance := (maxReasonableSpeed * timeDiff) / 3600 // km
			ratio := maxDistance / distance
			estimatedLoc := s.interpolateLocation(prevPoint.Location, currentPoint.Location, ratio)
			analysis.EstimatedLocation = &estimatedLoc

			// Update GPS accuracy metric (error in meters)
			accuracyMeters := distance * 1000 * (1 - analysis.Confidence)
			metrics.GpsAccuracy.WithLabelValues(driverID).Set(accuracyMeters)
		}

		analyzedPoints[i].Analysis = analysis
		analyzedPoints[i].Speed = analysis.CalculatedSpeed
		totalSpeed += speed
		if speed > maxSpeed && !analysis.IsAnomaly {
			maxSpeed = speed
		}
	}

	averageSpeed := totalSpeed / float64(len(points)-1)
	confidence := 1.0 - (float64(anomalyCount) / float64(len(points)))

	// Record metrics
	metrics.PointsInAnalysis.WithLabelValues(driverID).Observe(float64(len(points)))
	metrics.AnalysisLatency.WithLabelValues(driverID).Observe(time.Since(start).Seconds())

	span.SetAttributes(
		attribute.Float64("average_speed", averageSpeed),
		attribute.Float64("max_speed", maxSpeed),
		attribute.Float64("confidence", confidence),
		attribute.Int("anomaly_count", anomalyCount),
	)
	span.SetStatus(codes.Ok, "track analysis completed successfully")

	return &models.TrackAnalysis{
		DriverID:     driverID,
		Points:       analyzedPoints,
		AverageSpeed: averageSpeed,
		MaxSpeed:     maxSpeed,
		Confidence:   confidence,
		AnomalyCount: anomalyCount,
		Timestamp:    time.Now().Unix(),
	}
}

func (s *TrackService) calculateDistance(loc1, loc2 models.Location) float64 {
	lat1 := loc1.Latitude * math.Pi / 180
	lon1 := loc1.Longitude * math.Pi / 180
	lat2 := loc2.Latitude * math.Pi / 180
	lon2 := loc2.Longitude * math.Pi / 180

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c // Distance in kilometers
}

func (s *TrackService) interpolateLocation(start, end models.Location, ratio float64) models.Location {
	return models.Location{
		Latitude:  start.Latitude + (end.Latitude-start.Latitude)*ratio,
		Longitude: start.Longitude + (end.Longitude-start.Longitude)*ratio,
	}
}
