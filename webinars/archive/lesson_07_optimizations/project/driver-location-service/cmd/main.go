package main

import (
	"context"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"example/driver-location-service/internal/handlers"
	internalMiddleware "example/driver-location-service/internal/middleware"
	"example/driver-location-service/internal/service"
	"go.opentelemetry.io/contrib/bridges/otelslog"
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
			semconv.ServiceNameKey.String("driver-location-service"),
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

func initMeter() *metric.MeterProvider {
	exporter, err := otlpmetricgrpc.New(context.Background())
	if err != nil {
		slog.Error("failed to initialize meter", "error", err)
		os.Exit(1)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("driver-location-service"),
		)),
	)
	otel.SetMeterProvider(mp)

	return mp
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	tp := initTracer()
	defer tp.Shutdown(context.Background())

	mp := initMeter()
	defer mp.Shutdown(context.Background())

	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	defer redisClient.Close()

	driverService := service.NewDriverService(redisClient, os.Getenv("TRACK_ANALYZER_URL"), logger)
	driverHandler := handlers.NewDriverHandler(driverService, logger)

	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(internalMiddleware.TracingMiddleware)
	r.Use(internalMiddleware.MetricsMiddleware)

	ctx := context.Background()
	// Create the OTLP log exporter that sends logs to configured destination
	logExporter, err := otlploggrpc.New(ctx)
	if err != nil {
		panic("failed to initialize exporter")
	}

	// Create the logger provider
	lp := log.NewLoggerProvider(
		log.WithProcessor(
			log.NewBatchProcessor(logExporter),
		),
	)

	// Ensure the logger is shutdown before exiting so all pending logs are exported
	defer lp.Shutdown(ctx)

	// Set the logger provider globally
	global.SetLoggerProvider(lp)

	// Instantiate a new slog logger
	otelLogger := otelslog.NewLogger("my-otel-logger")
	otelLogger.Info("Example message from otel info")
	otelLogger.Error("Example message from otel error")
	otelLogger.Warn("Example message from otel warn")

	// Create a subrouter for API endpoints with timeout
	apiRouter := chi.NewRouter()
	apiRouter.Use(internalMiddleware.TimeoutMiddleware(time.Second))

	apiRouter.Route("/api/v1", func(r chi.Router) {
		r.Post("/drivers/{id}/location", driverHandler.UpdateLocation)
		r.Get("/drivers/nearby", driverHandler.FindNearbyDrivers)
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
