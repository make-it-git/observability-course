package service

import (
	"context"
	"example/track-analyzer-service/internal/tracing"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type FeatureValue interface{}

type FeatureService struct {
	flags sync.Map
}

func NewFeatureService() *FeatureService {
	return &FeatureService{}
}

func (s *FeatureService) SetFeature(ctx context.Context, name string, value FeatureValue) {
	_, span := tracing.StartSpan(ctx, "feature.set",
		attribute.String("feature_name", name),
		attribute.String("feature_type", getFeatureType(value)),
	)
	defer span.End()

	if value == nil {
		s.flags.Delete(name)
		span.SetStatus(codes.Ok, "feature deleted")
		return
	}
	s.flags.Store(name, value)
	span.SetStatus(codes.Ok, "feature set successfully")
}

func (s *FeatureService) GetBoolFeature(ctx context.Context, name string, defaultValue bool) bool {
	_, span := tracing.StartSpan(ctx, "feature.get_bool",
		attribute.String("feature_name", name),
		attribute.Bool("default_value", defaultValue),
	)
	defer span.End()

	value, ok := s.flags.Load(name)
	if !ok {
		span.SetStatus(codes.Ok, "feature not found, using default")
		return defaultValue
	}
	boolValue, ok := value.(bool)
	if !ok {
		span.SetStatus(codes.Ok, "feature type mismatch, using default")
		return defaultValue
	}
	span.SetStatus(codes.Ok, "feature retrieved successfully")
	return boolValue
}

func (s *FeatureService) GetIntFeature(ctx context.Context, name string, defaultValue int) int {
	_, span := tracing.StartSpan(ctx, "feature.get_int",
		attribute.String("feature_name", name),
		attribute.Int("default_value", defaultValue),
	)
	defer span.End()

	value, ok := s.flags.Load(name)
	if !ok {
		span.SetStatus(codes.Ok, "feature not found, using default")
		return defaultValue
	}
	intValue, ok := value.(int)
	if !ok {
		span.SetStatus(codes.Ok, "feature type mismatch, using default")
		return defaultValue
	}
	span.SetStatus(codes.Ok, "feature retrieved successfully")
	return intValue
}

func getFeatureType(value FeatureValue) string {
	if value == nil {
		return "nil"
	}
	switch value.(type) {
	case bool:
		return "bool"
	case int:
		return "int"
	default:
		return "unknown"
	}
}
