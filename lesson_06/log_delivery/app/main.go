package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	logger, loggerShutdown := getLogger(ctx)
	defer cancelCtx()
	defer loggerShutdown()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	apiRouter := chi.NewRouter()

	apiRouter.Route("/example", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			logger.Info("Request handled", "id", chi.URLParam(r, "id"), "personal_identifier", chi.URLParam(r, "id"))
			w.Write([]byte("OK"))
		})
	})
	r.Mount("/", apiRouter)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("starting service", "port", "8080")
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

func getLogger(ctx context.Context) (*slog.Logger, func()) {
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithRetry(otlploggrpc.RetryConfig{
		Enabled:         true,
		InitialInterval: time.Second,
		MaxInterval:     time.Second * 5,
		MaxElapsedTime:  time.Second * 30,
	}), otlploggrpc.WithReconnectionPeriod(time.Second*3))
	if err != nil {
		panic(err)
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", "development"),
		),
	)

	if err != nil {
		panic(err)
	}

	consoleExporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	lp := log.NewLoggerProvider(
		log.WithResource(r),
		log.WithProcessor(
			log.NewBatchProcessor(
				logExporter,
				log.WithMaxQueueSize(3),
				log.WithExportTimeout(time.Second),
				log.WithExportInterval(time.Second),
				log.WithExportMaxBatchSize(100),
			),
		),
		log.WithProcessor(log.NewSimpleProcessor(consoleExporter)),
	)

	// Set the logger provider globally
	global.SetLoggerProvider(lp)

	// Instantiate a new slog logger
	return otelslog.NewLogger("logger"), func() {
		lp.Shutdown(ctx)
		consoleExporter.Shutdown(ctx)
	}
}
