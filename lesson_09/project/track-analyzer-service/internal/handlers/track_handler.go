package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"example/track-analyzer-service/internal/domain/models"
	"example/track-analyzer-service/internal/metrics"
	"example/track-analyzer-service/internal/repository"
	"example/track-analyzer-service/internal/service"
)

var tracer = otel.Tracer("track-handlers")

type TrackHandler struct {
	repo           repository.TrackRepository
	logger         *slog.Logger
	featureService *service.FeatureService
	random         *rand.Rand
}

func NewTrackHandler(repo repository.TrackRepository, logger *slog.Logger, featureService *service.FeatureService) *TrackHandler {
	source := rand.NewSource(time.Now().UnixNano())
	return &TrackHandler{
		repo:           repo,
		logger:         logger,
		featureService: featureService,
		random:         rand.New(source),
	}
}

func (h *TrackHandler) AddPoint(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "AddPoint")
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	logger := h.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
	)

	driverID := chi.URLParam(r, "driverID")
	logger = logger.With(slog.String("driverID", driverID))

	var point models.GpsPoint
	if err := json.NewDecoder(r.Body).Decode(&point); err != nil {
		logger.Error("failed to decode point", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(point)
	if err != nil {
		logger.Error("failed to marshal point", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.repo.SavePoint(ctx, driverID, data); err != nil {
		logger.Error("failed to save point", "error", err)
		http.Error(w, "Failed to save point", http.StatusInternalServerError)
		return
	}

	metrics.ProcessedPoints.WithLabelValues(driverID).Inc()
	logger.Info("point processed successfully",
		"latitude", point.Location.Latitude,
		"longitude", point.Location.Longitude,
	)

	h.randomWait(ctx, logger)

	w.WriteHeader(http.StatusOK)
}

func (h *TrackHandler) GetRecentPoints(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "GetRecentPoints")
	defer span.End()

	spanCtx := trace.SpanContextFromContext(ctx)
	logger := h.logger.With(
		slog.String("traceID", spanCtx.TraceID().String()),
	)

	driverID := chi.URLParam(r, "driverID")
	logger = logger.With(slog.String("driverID", driverID))

	count := 50 // default count
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		var err error
		count, err = strconv.Atoi(countStr)
		if err != nil {
			logger.Error("invalid count parameter", "error", err)
			http.Error(w, "Invalid count parameter", http.StatusBadRequest)
			return
		}
		if count <= 0 || count > 1000 {
			http.Error(w, "Count must be between 1 and 1000", http.StatusBadRequest)
			return
		}
	}

	points, err := h.repo.GetRecentPoints(ctx, driverID, count)
	if err != nil {
		logger.Error("failed to get recent points", "error", err)
		http.Error(w, "Failed to get recent points", http.StatusInternalServerError)
		return
	}

	logger.Info("retrieved recent points", "count", len(points))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(points); err != nil {
		logger.Error("failed to encode response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *TrackHandler) randomWait(ctx context.Context, logger *slog.Logger) {
	min := h.featureService.GetIntFeature(ctx, "add-point-delay-value-start", 0)
	max := h.featureService.GetIntFeature(ctx, "add-point-delay-value-end", 0)

	s := 0.0
	if min > 0 && max > 0 && max > min {
		randomValue := min + h.random.Intn(max-min+1)
		logger.Info("Random sleep", "value", randomValue)
		time.Sleep(time.Millisecond * time.Duration(randomValue))
		for i := 0; i < 10_000_000; i++ {
			s = math.Max(float64(i), 100) // generate cpu load
		}
	}
	_ = s
}
