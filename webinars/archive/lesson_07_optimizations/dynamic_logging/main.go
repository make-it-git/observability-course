package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

var LOG_LEVEL = new(slog.LevelVar)

func main() {
	LOG_LEVEL.Set(slog.LevelInfo)

	opts := &slog.HandlerOptions{
		Level: LOG_LEVEL,
	}
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, opts),
	)

	go func() {
		for {
			time.Sleep(time.Second * 3)
			if LOG_LEVEL.Level() == slog.LevelInfo {
				fmt.Println("Set Debug")
				LOG_LEVEL.Set(slog.LevelDebug)
			} else {
				fmt.Println("Set Info")
				LOG_LEVEL.Set(slog.LevelInfo)
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second)
			logger.Info("Info")
			logger.Debug("Debug")
		}
	}()

	select {}
}
