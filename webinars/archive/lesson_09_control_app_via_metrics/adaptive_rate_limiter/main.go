package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	defaultPrometheusAddress    = "http://localhost:9090"
	defaultQueryIntervalSeconds = 3
	defaultInitialRateLimit     = 60
	defaultErrorRateThreshold   = 0.05 // 5% error rate
	defaultRateIncreaseFactor   = 1.1
	defaultRateDecreaseFactor   = 0.9
	defaultMaxRateLimit         = 60
	defaultMinRateLimit         = 1
	defaultTargetMetric         = "app_requests_total"
	defaultErrorMetric          = "app_requests_errors_total"
	defaultPromQLQuery          = "rate(%s[1m]) / rate(%s[1m])"
)

var (
	prometheusAddress    string
	queryIntervalSeconds int
	initialRateLimit     int
	errorRateThreshold   float64
	rateIncreaseFactor   float64
	rateDecreaseFactor   float64
	maxRateLimit         int
	minRateLimit         int
	targetMetric         string
	errorMetric          string
	promQLQuery          string
)

var (
	requestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_requests_total",
		Help: "Total number of requests.",
	})
	requestErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_requests_errors_total",
		Help: "Total number of request errors.",
	})
	rateLimitGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "app_rate_limit",
		Help: "Current rate limit.",
	})
)

type RateLimiter struct {
	rateLimitPerMinute int
	lastUpdated        time.Time
	mu                 sync.Mutex
	requestCount       int
	currentMinute      time.Time
	client             v1.API
}

func NewRateLimiter(client v1.API, initialRate int) *RateLimiter {
	return &RateLimiter{
		rateLimitPerMinute: initialRate,
		lastUpdated:        time.Now(),
		currentMinute:      time.Now().Truncate(time.Minute),
		client:             client,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	currentMinute := now.Truncate(time.Minute) // Get the beginning of the current minute

	// If the current minute is different from the tracked minute, reset the count
	if currentMinute != rl.currentMinute {
		rl.requestCount = 0
		rl.currentMinute = currentMinute
	}

	if rl.requestCount < rl.rateLimitPerMinute {
		rl.requestCount++
		log.Printf("Request allowed. Count: %d\n", rl.requestCount)
		return true
	} else {
		log.Println("Rate limit exceeded (for this minute)!")
		return false
	}
}

func (rl *RateLimiter) GetRateLimit() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.rateLimitPerMinute
}

func (rl *RateLimiter) SetRateLimit(rate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rateLimitPerMinute = rate
	rateLimitGauge.Set(float64(rate))
	log.Printf("Rate limit updated to: %d", rate)
}

func (rl *RateLimiter) MonitorAndAdjust(queryInterval time.Duration, errorRateThreshold float64, rateIncreaseFactor float64, rateDecreaseFactor float64, maxRateLimit int, minRateLimit int) {
	ticker := time.NewTicker(queryInterval)
	defer ticker.Stop()

	for range ticker.C {
		errorRate, err := rl.queryErrorRate()
		if err != nil {
			log.Printf("Error querying error rate: %v", err)
			continue
		}
		log.Printf("Current error rate: %f\n", errorRate)

		currentRateLimit := rl.GetRateLimit()

		if errorRate > errorRateThreshold {
			// Decrease rate limit if error rate is too high
			newRateLimit := int(math.Floor(float64(currentRateLimit) * rateDecreaseFactor))
			if newRateLimit < minRateLimit {
				newRateLimit = minRateLimit
			}
			rl.SetRateLimit(newRateLimit)
		} else {
			// Increase rate limit if error rate is acceptable
			newRateLimit := int(math.Ceil(float64(currentRateLimit) * rateIncreaseFactor))
			if newRateLimit > maxRateLimit {
				newRateLimit = maxRateLimit
			}
			rl.SetRateLimit(newRateLimit)
		}
	}
}

func (rl *RateLimiter) queryErrorRate() (float64, error) {
	ctx := context.Background()

	formattedQuery := fmt.Sprintf(promQLQuery, errorMetric, targetMetric)

	result, warnings, err := rl.client.Query(ctx, formattedQuery, time.Now())
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %v", warnings)
	}

	switch result.Type() {
	case model.ValVector:
		vectorValue := result.(model.Vector)
		if len(vectorValue) == 0 {
			return 0, nil
		}

		floatValue, err := strconv.ParseFloat(vectorValue[0].Value.String(), 64)
		if err != nil {
			return 0, fmt.Errorf("error converting query result: %w", err)
		}
		if math.IsNaN(floatValue) {
			return 0, nil
		}
		return floatValue, nil

	default:
		return 0, fmt.Errorf("unexpected result type: %s", result.Type())
	}
}

func main() {
	prometheusAddress = getEnv("PROMETHEUS_ADDRESS", defaultPrometheusAddress)
	queryIntervalSeconds = getEnvInt("QUERY_INTERVAL_SECONDS", defaultQueryIntervalSeconds)
	initialRateLimit = getEnvInt("INITIAL_RATE_LIMIT", defaultInitialRateLimit)
	errorRateThreshold = getEnvFloat("ERROR_RATE_THRESHOLD", defaultErrorRateThreshold)
	rateIncreaseFactor = getEnvFloat("RATE_INCREASE_FACTOR", defaultRateIncreaseFactor)
	rateDecreaseFactor = getEnvFloat("RATE_DECREASE_FACTOR", defaultRateDecreaseFactor)
	maxRateLimit = getEnvInt("MAX_RATE_LIMIT", defaultMaxRateLimit)
	minRateLimit = getEnvInt("MIN_RATE_LIMIT", defaultMinRateLimit)
	targetMetric = getEnv("TARGET_METRIC", defaultTargetMetric)
	errorMetric = getEnv("ERROR_METRIC", defaultErrorMetric)
	promQLQuery = getEnv("PROMQL_QUERY", defaultPromQLQuery)

	// Prometheus Client
	config := api.Config{
		Address: prometheusAddress,
	}
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatalf("Error creating Prometheus client: %v", err)
	}
	v1api := v1.NewAPI(client)

	rateLimiter := NewRateLimiter(v1api, initialRateLimit)
	rateLimitGauge.Set(float64(initialRateLimit))

	go rateLimiter.MonitorAndAdjust(
		time.Duration(queryIntervalSeconds)*time.Second,
		errorRateThreshold,
		rateIncreaseFactor,
		rateDecreaseFactor,
		maxRateLimit,
		minRateLimit,
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if rateLimiter.Allow() {
			fmt.Fprintln(w, "Request processed")
		} else {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintln(w, "Rate limit exceeded")
		}
	})

	http.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.Inc()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.Inc()
		requestErrors.Inc()
		w.WriteHeader(http.StatusInternalServerError)
	})

	http.Handle("/metrics", promhttp.Handler())

	log.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Error parsing environment variable %s: %v. Using default value %d.", key, err, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Error parsing environment variable %s: %v. Using default value %f.", key, err, defaultValue)
		return defaultValue
	}
	return value
}
