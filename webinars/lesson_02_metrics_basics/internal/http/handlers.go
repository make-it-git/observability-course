package http

import (
	"app/internal/app"
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// Handlers определяет HTTP-обработчики.
type Handlers struct {
	service *app.Service
	logger  *zap.Logger
}

// NewHandlers создает новые HTTP-обработчики.
func NewHandlers(service *app.Service, logger *zap.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger,
	}
}

// UpdateDriverLocationHandler обрабатывает запрос на обновление местоположения водителя.
func (h *Handlers) UpdateDriverLocationHandler(w http.ResponseWriter, r *http.Request) {
	driverID := r.URL.Query().Get("driver_id")
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if driverID == "" || latStr == "" || lonStr == "" {
		h.logger.Error("missing required query parameters")
		http.Error(w, "missing required query parameters", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		h.logger.Error("invalid latitude", zap.Error(err))
		http.Error(w, "invalid latitude", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		h.logger.Error("invalid longitude", zap.Error(err))
		http.Error(w, "invalid longitude", http.StatusBadRequest)
		return
	}

	h.service.UpdateDriverLocation(driverID, lat, lon)
	w.WriteHeader(http.StatusOK)
	h.logger.Info("driver location updated via http", zap.String("driver_id", driverID), zap.Float64("lat", lat), zap.Float64("lon", lon))
}

// FindNearestDriversHandler обрабатывает запрос на поиск ближайших водителей.
func (h *Handlers) FindNearestDriversHandler(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	limitStr := r.URL.Query().Get("limit")

	if latStr == "" || lonStr == "" || limitStr == "" {
		h.logger.Error("missing required query parameters")
		http.Error(w, "missing required query parameters", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		h.logger.Error("invalid latitude", zap.Error(err))
		http.Error(w, "invalid latitude", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		h.logger.Error("invalid longitude", zap.Error(err))
		http.Error(w, "invalid longitude", http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		h.logger.Error("invalid limit", zap.Error(err))
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return
	}
	h.logger.Info("find nearest drivers via http", zap.Float64("lat", lat), zap.Float64("lon", lon), zap.Int("limit", limit))
	drivers := h.service.FindNearestDrivers(lat, lon, limit)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]string{"drivers": drivers})
}
