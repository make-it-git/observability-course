package app

import (
	"app/internal/config"
	"app/internal/observability"
	"go.uber.org/zap"
)

// Service объединяет в себе все сервисы приложения.
type Service struct {
	locationManager *LocationManager
	searchService   *SearchService
	logger          *zap.Logger
}

// NewService создает новый Service.
func NewService(config *config.Config, metrics *observability.Metrics, logger *zap.Logger) *Service {
	locationManager := NewLocationManager(config, metrics)
	searchService := NewSearchService(locationManager)
	return &Service{
		locationManager: locationManager,
		searchService:   searchService,
		logger:          logger,
	}
}

// UpdateDriverLocation обновляет местоположение водителя.
func (s *Service) UpdateDriverLocation(driverID string, lat, lon float64) {
	s.locationManager.UpdateLocation(driverID, lat, lon)
	s.logger.Debug("Driver location updated", zap.String("driver_id", driverID), zap.Float64("lat", lat), zap.Float64("lon", lon))
}

// FindNearestDrivers ищет ближайших водителей к заданным координатам.
func (s *Service) FindNearestDrivers(lat, lon float64, limit int) []string {
	drivers := s.searchService.FindNearestDrivers(lat, lon, limit)
	s.logger.Debug("Nearest drivers found", zap.Float64("lat", lat), zap.Float64("lon", lon), zap.Int("limit", limit), zap.Strings("drivers", drivers))
	return drivers
}
