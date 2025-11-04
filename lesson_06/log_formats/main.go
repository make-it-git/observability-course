package main

import (
	"context"
	"log/slog"
	"os"
)

// go run main.go
// ENV=prod go run main.go | jq
func main() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var appEnv = os.Getenv("ENV")

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if appEnv == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	logger.LogAttrs(
		context.Background(),
		slog.LevelWarn,
		"User is a potential scammer",
		slog.Int("id", 12345),
		slog.String("scam_action", "withdraw_money"),
		slog.Group("properties",
			slog.Int("amount", 100500),
			slog.String("currency", "USD"),
		),
	)
}
