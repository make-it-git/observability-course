package handlers

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"example/track-analyzer-service/internal/service"
)

type FeatureHandler struct {
	featureService *service.FeatureService
	logger         *slog.Logger
}

func NewFeatureHandler(featureService *service.FeatureService, logger *slog.Logger) *FeatureHandler {
	return &FeatureHandler{
		featureService: featureService,
		logger:         logger,
	}
}

type FeatureRequest struct {
	Value interface{} `json:"value"`
}

func (h *FeatureHandler) SetFeature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	featureName := chi.URLParam(r, "name")

	var req FeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(ctx, "failed to decode request body",
			"error", err,
			"feature", featureName,
		)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert number to int if it's float64 (default JSON number type)
	if value, ok := req.Value.(float64); ok {
		req.Value = int(value)
	}

	h.featureService.SetFeature(ctx, featureName, req.Value)

	h.logger.InfoContext(ctx, "feature flag updated",
		"feature", featureName,
		"value", req.Value,
	)

	w.WriteHeader(http.StatusOK)
}
