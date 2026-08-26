package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"
)

// sampler holds state for sampling decisions.
type sampler struct {
	rate  float64 // Sampling rate (0.0 to 1.0)
	burst int     // Initial burst allowance

	mu    sync.Mutex // Protects state
	count int        // Number of messages since start.
}

func (s *sampler) allow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++

	// Allow initial burst
	if s.count <= s.burst {
		return true
	}

	// Sample based on rate. Use a simple random number comparison.
	return rand.Float64() < s.rate
}

// sampledHandler is a custom slog.Handler that implements sampling.
type sampledHandler struct {
	handler slog.Handler
	sampler *sampler
}

func (h *sampledHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *sampledHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.sampler.allow() {
		return h.handler.Handle(ctx, r)
	}
	// Writing to stderr for demo purposes only
	os.Stderr.Write([]byte(fmt.Sprintf("Ignore: %s\n", r.Message)))
	return nil
}

func (h *sampledHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sampledHandler{
		handler: h.handler.WithAttrs(attrs),
		sampler: h.sampler,
	}
}

func (h *sampledHandler) WithGroup(name string) slog.Handler {
	return &sampledHandler{
		handler: h.handler.WithGroup(name),
		sampler: h.sampler,
	}
}

// go run main.go 2>/dev/null
func main() {
	rand.Seed(time.Now().UnixNano())

	samp := &sampler{
		rate:  0.1, // Sample 10% of messages
		burst: 5,   // Allow first 5 messages
	}

	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	sampled := &sampledHandler{
		handler: baseHandler,
		sampler: samp,
	}
	logger := slog.New(sampled)
	slog.SetDefault(logger)

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		slog.InfoContext(ctx, fmt.Sprintf("Message %d", i))
		time.Sleep(10 * time.Millisecond) // Reduce rate for visibility.
	}
}
