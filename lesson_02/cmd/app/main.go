package main

import (
	"app/internal/app"
	"app/internal/config"
	handlers "app/internal/http"
	"app/internal/observability"
	"fmt"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	metrics := observability.NewMetrics()

	service := app.NewService(cfg, metrics, logger)

	handlers := handlers.NewHandlers(service, logger)

	http.HandleFunc("/drivers/update", handlers.UpdateDriverLocationHandler)
	http.HandleFunc("/drivers/search", handlers.FindNearestDriversHandler)

	http.Handle(cfg.MetricsPath, promhttp.Handler())

	logger.Info("starting server", zap.String("address", cfg.HTTPAddress))
	if err := http.ListenAndServe(cfg.HTTPAddress, nil); err != nil {
		logger.Error("failed to start server", zap.Error(err))
		os.Exit(1)
	}
}

func setupLogger(logLevel string) (*zap.Logger, error) {
	var config zap.Config

	if logLevel == "DEBUG" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(getLogLevel(logLevel))

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	return logger, nil
}

func getLogLevel(level string) zapcore.Level {
	switch level {
	case "DEBUG":
		return zap.DebugLevel
	case "INFO":
		return zap.InfoLevel
	case "WARN":
		return zap.WarnLevel
	case "ERROR":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}
