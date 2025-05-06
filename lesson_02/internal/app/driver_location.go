package app

import (
	"app/internal/config"
	"app/internal/observability"
	"app/internal/utils"
	"fmt"
	"sort"
	"sync"
	"time"
)

// DriverLocation содержит информацию о местоположении водителя.
type DriverLocation struct {
	Latitude  float64
	Longitude float64
	Timestamp time.Time
}

// LocationManager управляет местоположениями водителей.
type LocationManager struct {
	locations            map[string]DriverLocation
	bucketSize           float64
	positionStaleTimeout time.Duration
	mu                   sync.RWMutex
	config               *config.Config
	metrics              *observability.Metrics
}

// NewLocationManager создает новый LocationManager.
func NewLocationManager(config *config.Config, metrics *observability.Metrics) *LocationManager {
	lm := &LocationManager{
		locations:            make(map[string]DriverLocation),
		bucketSize:           config.BucketSize,
		positionStaleTimeout: config.PositionStaleTimeout,
		config:               config,
		metrics:              metrics,
	}
	go lm.cleanupOldPositions()
	return lm
}

// getBucket возвращает идентификатор бакета для заданных координат.
func (lm *LocationManager) getBucket(lat, lon float64) string {
	bucketLat := int(lat / lm.bucketSize)
	bucketLon := int(lon / lm.bucketSize)
	return fmt.Sprintf("%d,%d", bucketLat, bucketLon)
}

// UpdateLocation обновляет местоположение водителя.
func (lm *LocationManager) UpdateLocation(driverID string, lat, lon float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.locations[driverID] = DriverLocation{
		Latitude:  lat,
		Longitude: lon,
		Timestamp: time.Now(),
	}
	lm.metrics.DriverLocationUpdates.Inc()
	lm.metrics.DriverPositionUpdates.WithLabelValues(driverID).Inc()
}

// GetLocation получает местоположение водителя.
func (lm *LocationManager) GetLocation(driverID string) (DriverLocation, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	loc, ok := lm.locations[driverID]
	return loc, ok
}

// GetDriversInBucket возвращает список идентификаторов водителей в заданном бакете.
func (lm *LocationManager) GetDriversInBucket(lat, lon float64) []string {
	bucket := lm.getBucket(lat, lon)
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var drivers []string
	for driverID, loc := range lm.locations {
		if lm.getBucket(loc.Latitude, loc.Longitude) == bucket {
			drivers = append(drivers, driverID)
		}
	}
	return drivers
}

// GetAllLocations возвращает все местоположения водителей.
func (lm *LocationManager) GetAllLocations() map[string]DriverLocation {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	locations := make(map[string]DriverLocation, len(lm.locations))
	for k, v := range lm.locations {
		locations[k] = v
	}
	return locations
}

// cleanupOldPositions периодически удаляет устаревшие местоположения водителей.
func (lm *LocationManager) cleanupOldPositions() {
	ticker := time.NewTicker(lm.config.CleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		lm.cleanup()
	}
}

func (lm *LocationManager) cleanup() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	for driverID, loc := range lm.locations {
		if now.Sub(loc.Timestamp) > lm.positionStaleTimeout {
			delete(lm.locations, driverID)
			lm.metrics.DriverPositionsEvicted.Inc()
		}
	}
	lm.metrics.CurrentDrivers.Set(float64(len(lm.locations)))

}

// FindNearestDrivers ищет ближайших водителей к заданным координатам.
func (lm *LocationManager) FindNearestDrivers(lat, lon float64, limit int) []string {
	lm.metrics.FindNearestRequests.Inc()
	start := time.Now()
	defer func() {
		lm.metrics.FindNearestDuration.Observe(time.Since(start).Seconds())
		lm.metrics.FindNearestDurationSummary.Observe(time.Since(start).Seconds())
	}()

	drivers := lm.GetDriversInBucket(lat, lon)

	// Вычисляем расстояния и сортируем
	type DriverDistance struct {
		ID       string
		Distance float64
	}

	var distances []DriverDistance
	for _, driverID := range drivers {
		loc, ok := lm.GetLocation(driverID)
		if !ok {
			continue
		}
		distance := utils.Distance(lat, lon, loc.Latitude, loc.Longitude)
		distances = append(distances, DriverDistance{ID: driverID, Distance: distance})
	}

	if len(distances) == 0 {
		return []string{}
	}

	// Сортировка по расстоянию
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})

	if len(distances) > limit {
		distances = distances[:limit]
	}

	result := make([]string, len(distances))
	for i, d := range distances {
		result[i] = d.ID
	}

	lm.metrics.NearestDriversFound.Observe(float64(len(result)))

	return result
}
