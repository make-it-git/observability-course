package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

// Metrics содержит все метрики приложения.
type Metrics struct {
	// Общие метрики
	DriverLocationUpdates  prometheus.Counter // Счетчик обновлений местоположения водителей
	DriverPositionsEvicted prometheus.Counter // Счетчик удаленных устаревших позиций
	CurrentDrivers         prometheus.Gauge   // Количество текущих водителей

	// Метрики поиска
	FindNearestRequests        prometheus.Counter   // Счетчик запросов на поиск ближайших водителей
	FindNearestDuration        prometheus.Histogram // Гистограмма времени поиска ближайших водителей
	FindNearestDurationSummary prometheus.Summary   // Summary времени поиска ближайших водителей
	NearestDriversFound        prometheus.Histogram // Гистограмма количества найденных водителей

	// Бизнес метрики
	TaxiOrdersCreated prometheus.Counter // Счетчик созданных заказов такси

	// Пример метрики с высокой кардинальностью (не рекомендуется в production)
	DriverPositionUpdates *prometheus.CounterVec // Пример плохой метрики: счетчик обновлений координат водителя по ID
}

// NewMetrics создает и регистрирует новые метрики.
func NewMetrics() *Metrics {
	m := &Metrics{
		// Общие метрики
		DriverLocationUpdates: promauto.NewCounter(prometheus.CounterOpts{
			Name: "taxi_driver_location_updates_total",
			Help: "Total number of driver location updates",
		}),
		DriverPositionsEvicted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "taxi_driver_positions_evicted_total",
			Help: "Total number of stale driver positions evicted",
		}),
		CurrentDrivers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "taxi_driver_current_drivers",
			Help: "Current number of drivers",
		}),

		// Метрики поиска
		FindNearestRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "taxi_find_nearest_requests_total",
			Help: "Total number of find nearest driver requests",
		}),
		FindNearestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "taxi_find_nearest_duration_seconds",
			Help:    "Duration of find nearest driver requests",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		}),
		NearestDriversFound: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "taxi_nearest_drivers_found",
			Help:    "Number of nearest drivers found",
			Buckets: []float64{0, 1, 2, 3, 5, 10, 20},
		}),
		FindNearestDurationSummary: promauto.NewSummary(prometheus.SummaryOpts{
			Name: "taxi_find_nearest_duration_summary_seconds",
			Help: "Duration of find nearest driver requests as summary",
			Objectives: map[float64]float64{
				0.5:  0.05,  // 50% с погрешностью 5%
				0.9:  0.01,  // 90% с погрешностью 1%
				0.99: 0.001, // 99% с погрешностью 0.1%
			},
			MaxAge: time.Second * 10,
		}),

		// Бизнес метрики
		TaxiOrdersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "taxi_orders_created_total",
			Help: "Total number of taxi orders created",
		}),

		// Пример метрики с высокой кардинальностью
		DriverPositionUpdates: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "taxi_driver_position_updates",
			Help: "Total number of driver location updates by driver ID (HIGH CARDINALITY)",
		}, []string{"driver_id"}),
	}
	return m
}
