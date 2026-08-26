package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"example/driver-location-service/internal/domain/models"
	"example/driver-location-service/internal/metrics"
	"example/driver-location-service/internal/service"
)

var (
	tracer = otel.Tracer("driver-handlers")
	meter  = otel.GetMeterProvider().Meter("driver-handlers")
)

type DriverHandler struct {
	driverService   service.DriverService
	logger          *slog.Logger
	locationUpdates metric.Int64Counter
}

func NewDriverHandler(driverService service.DriverService, logger *slog.Logger) *DriverHandler {
	locationUpdates, err := meter.Int64Counter(
		"driver.location.updates",
		metric.WithDescription("Number of driver location updates"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Error("failed to create location updates counter", "error", err)
	} else {
		logger.Info("created otel counter driver.location.updates")
	}

	return &DriverHandler{
		driverService:   driverService,
		logger:          logger,
		locationUpdates: locationUpdates,
	}
}

func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "UpdateDriverLocation")
	defer span.End()

	driverID := chi.URLParam(r, "id")
	driverIDInt, _ := strconv.Atoi(driverID)
	span.SetAttributes(
		attribute.Int("driver_id", driverIDInt),
	)

	spanCtx := trace.SpanContextFromContext(ctx)
	span.SetStatus(codes.Ok, "track analysis completed successfully")
	logger := h.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
		slog.String("driverID", driverID),
	)

	logger = logger.With(slog.String("driverID", driverID))

	var location models.Location
	if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
		logger.Error("failed to decode location", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to decode location")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.driverService.UpdateLocation(context.WithoutCancel(ctx), driverID, location); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update location")
		logger.Error("failed to update location", "error", err)
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}

	// Record metrics
	h.locationUpdates.Add(ctx, 1, metric.WithAttributes(
		attribute.String("driver_id", driverID),
	))
	logger.Info("Updated otel metric driver.location.updates")

	metrics.LocationUpdates.WithLabelValues(driverID).Inc()

	logger.Info("location updated successfully",
		"latitude", location.Latitude,
		"longitude", location.Longitude,
	)

	span.SetStatus(codes.Ok, "Driver location updated")
	w.WriteHeader(http.StatusOK)
}

func (h *DriverHandler) FindNearbyDrivers(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "FindNearbyDrivers")
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	logger := h.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
	)

	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		logger.Error("invalid latitude", "error", err)
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		logger.Error("invalid longitude", "error", err)
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		return
	}

	// Default radius 5km
	radius := 5000.0
	if rad := r.URL.Query().Get("radius"); rad != "" {
		radius, err = strconv.ParseFloat(rad, 64)
		if err != nil {
			logger.Error("invalid radius", "error", err)
			http.Error(w, "Invalid radius", http.StatusBadRequest)
			return
		}
	}

	logger = logger.With(
		slog.Float64("latitude", lat),
		slog.Float64("longitude", lon),
		slog.Float64("radius", radius),
	)

	drivers, err := h.driverService.FindNearby(ctx, lat, lon, radius)
	if err != nil {
		logger.Error("failed to find nearby drivers", "error", err)
		http.Error(w, "Failed to find nearby drivers", http.StatusInternalServerError)
		return
	}

	logger.Info("found nearby drivers", "count", len(drivers))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(drivers); err != nil {
		logger.Error("failed to encode response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
