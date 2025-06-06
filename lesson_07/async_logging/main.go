package main

import (
	"fmt"
	"log/slog"
	"os"
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
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

// W worker that processes messages from the channel
func asyncLogWorker() {
	defer wg.Done()
	for msg := range logChannel {
		logger.Info(msg)
	}
}

func asyncLog(message string) {
	select {
	case logChannel <- message: // Blocking send
	}
	//select {
	//case logChannel <- message: // Non-blocking send
	//default:
	//	os.Stderr.Write([]byte("Message lost\n"))
	//	// Prevent blocking when the channel is full
	//}
}

func generateLogMessage(i int) string {
	return fmt.Sprintf("Logging %d", i)
}

// Faster
// time SIMPLE_LOGGING=1 go run main.go > /dev/null
// SIMPLE_LOGGING=1 go run main.go > /dev/null  2.54s user 1.26s system 101% cpu 3.746 total

// Slower
// time go run main.go > /dev/null
// go run main.go > /dev/null  5.20s user 3.11s system 173% cpu 4.777 total
func main() {

	const N = 5_000_000
	if os.Getenv("SIMPLE_LOGGING") == "1" {
		os.Stderr.Write([]byte(fmt.Sprintf("Simple started %v\n", time.Now())))
		for i := 0; i < N; i++ {
			logger.Info(generateLogMessage(i))
		}
		os.Stderr.Write([]byte(fmt.Sprintf("Simple completed %v\n", time.Now())))
	} else {
		os.Stderr.Write([]byte(fmt.Sprintf("Async started %v\n", time.Now())))
		wg.Add(1)
		go asyncLogWorker()
		for i := 0; i < N; i++ {
			asyncLog(generateLogMessage(i))
		}
		os.Stderr.Write([]byte(fmt.Sprintf("Async generated all log messages %v\n", time.Now())))
		close(logChannel)
		os.Stderr.Write([]byte(fmt.Sprintf("Async waiting to complete %v\n", time.Now())))
		wg.Wait()
		os.Stderr.Write([]byte(fmt.Sprintf("Async started %v\n", time.Now())))
	}
}
