package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
)

func initProvider() *sdkmetric.MeterProvider {
	// Создаем Prometheus экспортер.
	// Этот экспортер будет предоставлять метрики на /metrics endpoint.
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("failed to create prometheus exporter: %v", err)
	}

	// Определяем ресурс, который описывает сервис, генерирующий телеметрию.
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("my-golang-service-otel"),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Создаем MeterProvider с экспортером и ресурсом.
	// MeterProvider - это основной компонент для управления метриками.
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
		// Здесь можно добавить Views для фильтрации атрибутов.
		// Например, для контроля кардинальности.
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "http_request_duration_seconds"},
			sdkmetric.Stream{AttributeFilter: attribute.NewAllowKeysFilter(attribute.Key("path"), attribute.Key("method"))},
		)),
	)
	// Устанавливаем глобальный MeterProvider.
	// Это позволяет получать Meter через otel.Meter() в любом месте приложения.
	otel.SetMeterProvider(mp)

	// Получаем Meter для создания инструментов метрик.
	meter := otel.Meter("my-app-meter")

	// Регистрируем метрики.
	requestCounter, err = meter.Int64Counter("http_requests_total",
		metric.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		log.Fatalf("failed to create counter: %v", err)
	}

	requestDuration, err = meter.Float64Histogram("http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests."),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Fatalf("failed to create histogram: %v", err)
	}

	activeRequests, err = meter.Int64UpDownCounter("http_active_requests",
		metric.WithDescription("Number of active HTTP requests."),
	)
	if err != nil {
		log.Fatalf("failed to create updowncounter: %v", err)
	}

	return mp
}

// while true; do curl localhost:2112/hello; done
// while true; do curl -s localhost:2112/metrics | grep -E '^http_active'; sleep 0.2; done
//
// Filtered attributes
// while true; do curl -s localhost:2112/metrics | grep -E '^http_request_duration_seconds_count'; sleep 0.2; done
// Unfiltered
// while true; do curl -s localhost:2112/metrics | grep -E '^http_requests_total'; sleep 0.2; done
func main() {
	ctx := context.Background()
	mp := initProvider()
	defer func() {
		// Важно корректно завершить MeterProvider при завершении приложения.
		if err := mp.Shutdown(ctx); err != nil {
			log.Fatalf("failed to shutdown MeterProvider: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())

	// HTTP handler для имитации работы сервиса.
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// Увеличиваем счетчик активных запросов.
		activeRequests.Add(ctx, 1, metric.WithAttributes(
			attribute.String("path", "/hello"),
			attribute.String("method", r.Method),
			attribute.String("status_code", "200"),
		))
		defer activeRequests.Add(ctx, -1, metric.WithAttributes(
			attribute.String("path", "/hello"),
			attribute.String("method", r.Method),
			attribute.String("status_code", "200"),
		)) // Уменьшаем при выходе из функции.

		start := time.Now()
		// Имитация работы с случайной задержкой.
		time.Sleep(time.Duration(rand.Intn(500)+50) * time.Millisecond)

		// Записываем метрики. Атрибуты добавляют контекст.
		requestCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("path", "/hello"),
			attribute.String("method", r.Method),
			attribute.String("status_code", "200"), // Пример атрибута статуса
		))
		requestDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("path", "/hello"),
			attribute.String("method", r.Method),
			attribute.String("status_code", "200"),
		))

		w.Write([]byte("Hello, OpenTelemetry!"))
	})

	log.Println("Serving metrics at :2112/metrics")
	log.Fatal(http.ListenAndServe(":2112", nil)) // Запускаем HTTP сервер для метрик.
}
