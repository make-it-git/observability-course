package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logger     *slog.Logger
	logChannel = make(chan string, 100_000) // Buffered channel
	wg         sync.WaitGroup
)

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

// worker that processes messages from the channel in batches
func asyncLogBatchWorker() {
	defer wg.Done()
	const BATCH_SIZE = 1000
	builder := strings.Builder{}
	index := 0
	for msg := range logChannel {
		index += 1
		builder.Write([]byte(msg))
		if index >= BATCH_SIZE {
			os.Stdout.Write([]byte(builder.String()))
			index = 0
			builder.Reset()
		}
	}
}

func asyncLogBatch(message string) {
	select {
	case logChannel <- message: // Blocking send
	}
}

func generateLogMessage(i int) string {
	return fmt.Sprintf("Logging %d\n", i)
}

// time SIMPLE_LOGGING=1 go run main.go > simple.log
// SIMPLE_LOGGING=1 go run main.go > simple.log  1.15s user 4.99s system 100% cpu 6.087 total
// wc -l simple.log
// 5000000 simple.log

// time go run main.go > async_batch.log
// go run main.go > async_batch.log  1.15s user 0.66s system 157% cpu 1.151 total
// wc -l async_batch.log
// 5000000 async_batch.log

func main() {
	const N = 5_000_000
	if os.Getenv("SIMPLE_LOGGING") == "1" {
		os.Stderr.Write([]byte(fmt.Sprintf("Simple started %v\n", time.Now())))
		for i := 0; i < N; i++ {
			os.Stdout.Write([]byte(generateLogMessage(i)))
		}
		os.Stderr.Write([]byte(fmt.Sprintf("Simple completed %v\n", time.Now())))
	} else {
		os.Stderr.Write([]byte(fmt.Sprintf("Async started %v\n", time.Now())))
		wg.Add(1)
		go asyncLogBatchWorker()
		for i := 0; i < N; i++ {
			asyncLogBatch(generateLogMessage(i))
		}
		os.Stderr.Write([]byte(fmt.Sprintf("Async generated all log messages %v\n", time.Now())))
		close(logChannel)
		os.Stderr.Write([]byte(fmt.Sprintf("Async waiting to complete %v\n", time.Now())))
		wg.Wait()
		os.Stderr.Write([]byte(fmt.Sprintf("Async started %v\n", time.Now())))
	}
}
