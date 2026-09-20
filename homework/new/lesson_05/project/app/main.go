package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	_ "go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
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
			userID := chi.URLParam(r, "id")
			result, err := handleRequest(region, userID)
			if err != nil {
				slog.ErrorContext(ctx, "Request failed", "path", "/user/{userID}", "userID", userID, "region", region)
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			slog.InfoContext(ctx, "Request handled", "path", "/user/{userID}", "userID", userID, "region", region)
			w.Write([]byte(result))
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

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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

	// Uncomment to see otel logs in console
	// consoleExporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
	// if err != nil {
	// 	panic(err)
	// }

	lp := log.NewLoggerProvider(
		log.WithResource(r),
		log.WithProcessor(
			log.NewBatchProcessor(
				logExporter,
			),
		),
		//  Uncomment to see otel logs in console
		// log.WithProcessor(log.NewSimpleProcessor(consoleExporter)),
	)

	// Set the logger provider globally
	global.SetLoggerProvider(lp)

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	otelHandler := otelslog.NewLogger("app")
	multiHandler := slogmulti.Fanout(jsonHandler, otelHandler.Handler())
	logger := slog.New(multiHandler)

	slog.SetDefault(logger)

	return func() {
		lp.Shutdown(ctx)
	}
}

// Do not scroll below this line unless you have built required dashboard as was stated in homework
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

func handleRequest(region string, userID string) (string, error) {
	if region == "VN" {
		if rand.Intn(100) > 95 {
			return "", errors.New("unexpected error")
		}
	}

	if region == "ZA" {
		if rand.Intn(100) > 90 {
			return "", errors.New("unexpected error")
		}
	}

	if region == "IN" {
		if rand.Intn(100) > 80 {
			return "", errors.New("unexpected error")
		}
	}

	return fmt.Sprintf("data for region %s for user %s: dummy data", region, userID), nil
}
