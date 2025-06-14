package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	logger     *slog.Logger
	logChannel = make(chan string, 100_000) // Buffered channel
	wg         sync.WaitGroup
	seenLogs   = make(map[string]int) // Map to track log message counts
	seenLogsMu sync.Mutex             // Mutex to protect access to seenLogs
)

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

// worker that processes messages from the channel in batches
func aggregateLogWorker() {
	defer wg.Done()
	const BATCH_SIZE = 1000
	builder := strings.Builder{}
	index := 0
	for msg := range logChannel {
		seenLogsMu.Lock()
		seenLogs[msg]++
		seenLogsMu.Unlock()
		index += 1
		if index >= BATCH_SIZE {
			flushLogs(builder)
			index = 0
			builder.Reset()
		}

	}
	flushLogs(builder) // Flush remaining logs after channel is closed

}

func flushLogs(builder strings.Builder) {
	seenLogsMu.Lock()
	defer seenLogsMu.Unlock()
	for msg, count := range seenLogs {
		builder.WriteString(fmt.Sprintf("Log Message: %sCount: %d\n", msg, count))
		delete(seenLogs, msg) // Clear the processed logs to avoid duplicates on next flush
	}
	os.Stdout.Write([]byte(builder.String()))
}

func asyncLogBatch(message string) {
	select {
	case logChannel <- message: // Blocking send
	}
}

func generateLogMessage(i int) string {
	return fmt.Sprintf("Logging %d\n", i%5)
}

// go run main.go

func main() {
	const N = 1000
	wg.Add(1)
	go aggregateLogWorker()
	for i := 0; i < N; i++ {
		asyncLogBatch(generateLogMessage(i))
	}
	close(logChannel)
	wg.Wait()
}
