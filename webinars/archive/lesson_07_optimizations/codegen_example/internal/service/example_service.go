package service

import (
	"context"
	"log/slog"
)

//go:generate gowrap gen -g -p ./ -i ExampleService -t prometheus -o example_service_with_metrics.go
type ExampleService interface {
	ExampleMethod(ctx context.Context, param string) error
}

type exampleService struct {
	logger *slog.Logger
}

func NewExampleService(logger *slog.Logger) ExampleService {
	return &exampleService{
		logger: logger,
	}
}

func (s *exampleService) ExampleMethod(ctx context.Context, param string) error {
	logger := s.logger.With(
		slog.String("param", param),
	)
	logger.InfoContext(ctx, "Doing something")
	return nil
}
