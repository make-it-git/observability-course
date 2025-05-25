package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	loggerShutdown := setupLogger(ctx)
	defer cancelCtx()
	defer loggerShutdown()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	apiRouter := chi.NewRouter()

	apiRouter.Route("/user", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			region := r.Header.Get("X-Region")
			if region == "VN" || region == "ZA" {
				if rand.Intn(100) > 90 {
					slog.ErrorContext(ctx, "Request failed", "userID", chi.URLParam(r, "id"), "region", region)
					w.WriteHeader(500)
					w.Write([]byte("KO"))
					return
				}
			}
			slog.InfoContext(ctx, "Request handled", "path", "/user/{userID}", "userID", chi.URLParam(r, "id"), "region", region)
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
		slog.InfoContext(ctx, "starting service", "port", "8080")
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		slog.ErrorContext(ctx, "server error", "error", err)
		return

	case sig := <-shutdown:
		slog.InfoContext(ctx, "shutdown started", "signal", sig)
		defer slog.InfoContext(ctx, "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "graceful shutdown failed", "error", err)
			if err := srv.Close(); err != nil {
				slog.ErrorContext(ctx, "forcing server close failed", "error", err)
			}
		}
	}
}

func setupLogger(ctx context.Context) func() {
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

	/*
		Otel can send details to console,
		but it will clutter console and will make it unusable.
		Feel free to uncomment to see in action.
		consoleExporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
		if err != nil {
			panic(err)
		}*/

	lp := log.NewLoggerProvider(
		log.WithResource(r),
		log.WithProcessor(
			log.NewBatchProcessor(
				logExporter,
			),
		),
		//log.WithProcessor(log.NewSimpleProcessor(consoleExporter)),
	)

	// Set the logger provider globally
	global.SetLoggerProvider(lp)

	slog.SetDefault(otelslog.NewLogger("main"))

	return func() {
		lp.Shutdown(ctx)
	}
}
