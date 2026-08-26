package main

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
docker compose up -d

OTEL_SERVICE_NAME="my-service" \
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317" \
OTEL_EXPORTER_OTLP_INSECURE=true \
go run main.go
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

	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	consoleTraceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSyncer(consoleTraceExporter),
		sdktrace.WithResource(r),
	)
	defer tp.Shutdown(ctx)
	otel.SetTracerProvider(tp)

	// Instantiate a new slog logger
	logger := otelslog.NewLogger("logger")
	tracer := otel.Tracer("tracer1")
	tracer2 := otel.Tracer("tracer2")

	// We can replace default logger with otel
	//slog.SetDefault(logger)

	// You can use the logger directly anywhere in your app now
	for i := 1; i <= 10; i++ {
		_, innerSpan := tracer.Start(ctx, "Iteration")
		_, innerSpan2 := tracer2.Start(ctx, "Iteration")
		logger.Info("Otel", "i", i)
		slog.Info("Slog", "i", i)

		innerSpan.AddEvent("Processing iteration", trace.WithAttributes(attribute.Int("i", i)))
		innerSpan.End()

		innerSpan2.AddEvent("Processing iteration", trace.WithAttributes(attribute.Int("i", i)))
		innerSpan2.End()
	}

	time.Sleep(time.Hour)
}
