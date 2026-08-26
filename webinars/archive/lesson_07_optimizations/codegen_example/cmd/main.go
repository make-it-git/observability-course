package main

import (
	"context"
	handlers "example/internal/handlers"
	service "example/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	exampleService := service.NewExampleService(logger)
	exampleServiceProm := service.NewExampleServiceWithPrometheus(exampleService, "main")
	exampleHandler := handlers.NewExampleHandler(exampleServiceProm, logger)

	r := chi.NewRouter()
	apiRouter := chi.NewRouter()

	apiRouter.Route("/api/v1", func(r chi.Router) {
		r.Get("/example/{param}", exampleHandler.DoSomething)
	})
	r.Mount("/", apiRouter)

	opts := promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}
	r.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, opts))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("starting driver location service", "port", "8080")
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		logger.Error("server error", "error", err)
		return

	case sig := <-shutdown:
		logger.Info("shutdown started", "signal", sig)
		defer logger.Info("shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			if err := srv.Close(); err != nil {
				logger.Error("forcing server close failed", "error", err)
			}
		}
	}
}
