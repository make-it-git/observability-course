package main

import (
	"context"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
)

/*
OTEL_SERVICE_NAME="my-service" \
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317" \
OTEL_EXPORTER_OTLP_INSECURE=true \
go run main.go

docker compose up -d
*/

func main() {
	ctx := context.Background()
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

	// Ensure the logger is shutdown before exiting so all pending logs are exported
	defer lp.Shutdown(ctx)
	defer consoleExporter.Shutdown(ctx)

	// Set the logger provider globally
	global.SetLoggerProvider(lp)

	// Instantiate a new slog logger
	logger := otelslog.NewLogger("logger")

	// We can replace default logger with otel
	//slog.SetDefault(logger)

	// You can use the logger directly anywhere in your app now
	for i := 1; i <= 10; i++ {
		logger.Info("Otel", "i", i)
		slog.Info("Slog", "i", i)
	}

	time.Sleep(time.Hour)
}
