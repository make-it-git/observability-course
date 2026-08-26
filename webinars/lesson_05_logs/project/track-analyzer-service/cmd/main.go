package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/grafana/pyroscope-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"example/track-analyzer-service/internal/handlers"
	internalMiddleware "example/track-analyzer-service/internal/middleware"
	"example/track-analyzer-service/internal/repository"
	"example/track-analyzer-service/internal/service"
)

func initTracer() *sdktrace.TracerProvider {
	exporter, err := otlptracegrpc.New(context.Background())
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("track-analyzer-service"),
		)),
	)
	otel.SetTracerProvider(tp)

	// Configure the W3C trace context propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	// Initialize Pyroscope
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "track-analyzer-service",
		ServerAddress:   os.Getenv("PYROSCOPE_SERVER_ADDRESS"),
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileGoroutines,
		},
	})

	tp := initTracer()
	defer tp.Shutdown(context.Background())

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	defer redisClient.Close()

	featureService := service.NewFeatureService()
	trackRepo := repository.NewRedisTrackRepository(redisClient, featureService)

	trackHandler := handlers.NewTrackHandler(trackRepo, logger, featureService)
	featureHandler := handlers.NewFeatureHandler(featureService, logger)

	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(internalMiddleware.TracingMiddleware)
	r.Use(internalMiddleware.MetricsMiddleware)

	// Create a subrouter for API endpoints with timeout
	apiRouter := chi.NewRouter()
	apiRouter.Use(internalMiddleware.TimeoutMiddleware(time.Second))

	apiRouter.Route("/api/v1", func(r chi.Router) {
		r.Post("/tracks/{driverID}/points", trackHandler.AddPoint)
		r.Get("/tracks/{driverID}/points", trackHandler.GetRecentPoints)
		r.Route("/features", func(r chi.Router) {
			r.Put("/{name}", featureHandler.SetFeature)
		})
	})

	// Mount the API router under the main router
	r.Mount("/", apiRouter)

	opts := promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}
	r.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, opts))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)
	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start the service listening for requests.
	go func() {
		logger.Info("starting track analyzer service", "port", "8081")
		serverErrors <- srv.ListenAndServe()
	}()

	// Blocking main and waiting for shutdown.
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
