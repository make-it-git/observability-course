package handlers

import (
	"example/internal/service"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
)

var (
	tracer = otel.Tracer("driver-handlers")
	meter  = otel.GetMeterProvider().Meter("driver-handlers")
)

type ExampleHandler struct {
	exampleService service.ExampleService
	logger         *slog.Logger
}

func NewExampleHandler(exampleService service.ExampleService, logger *slog.Logger) *ExampleHandler {
	return &ExampleHandler{
		exampleService: exampleService,
		logger:         logger,
	}
}

func (h *ExampleHandler) DoSomething(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "param")
	ctx := r.Context()
	if err := h.exampleService.ExampleMethod(ctx, param); err != nil {
		h.logger.Error("failed to handle", "error", err)
		http.Error(w, "Failed to handle", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("ok"))
	w.WriteHeader(http.StatusOK)
}
