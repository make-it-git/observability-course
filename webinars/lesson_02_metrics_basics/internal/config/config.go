package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	LogLevel             string        `envconfig:"LOG_LEVEL" default:"INFO"`
	HTTPAddress          string        `envconfig:"HTTP_ADDRESS" default:"0.0.0.0:8080"`
	MetricsPath          string        `envconfig:"METRICS_PATH" default:"/metrics"`
	BucketSize           float64       `envconfig:"BUCKET_SIZE" default:"0.01"`           // Размер бакета в градусах
	PositionStaleTimeout time.Duration `envconfig:"POSITION_STALE_TIMEOUT" default:"30s"` // Время протухания позиции водителя
	CleanupInterval      time.Duration `envconfig:"CLEANUP_INTERVAL" default:"5m"`        // Интервал очистки устаревших позиций
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
