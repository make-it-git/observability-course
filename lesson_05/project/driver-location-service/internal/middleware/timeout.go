package middleware

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"time"
)

// TimeoutMiddleware wraps an http.Handler and enforces a timeout for all requests
func TimeoutMiddleware(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal when the request is done
			done := make(chan struct{})

			// Create a response writer that can detect if headers were written
			tw := &timeoutWriter{
				w: w,
			}

			// Execute the handler in a goroutine
			go func() {
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			// Wait for either the request to complete or timeout
			select {
			case <-done:
				// Request completed normally
				return
			case <-ctx.Done():
				// Request timed out
				if !tw.headerWritten {
					w.WriteHeader(http.StatusGatewayTimeout)
					w.Write([]byte("Request timeout"))
				}
				return
			}
		})
	}
}

// timeoutWriter wraps an http.ResponseWriter and tracks if headers were written
type timeoutWriter struct {
	w             http.ResponseWriter
	headerWritten bool
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.headerWritten = true
	return tw.w.Write(b)
}

func (tw *timeoutWriter) WriteHeader(statusCode int) {
	tw.headerWritten = true
	tw.w.WriteHeader(statusCode)
}

// Hijack implements the http.Hijacker interface
func (tw *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := tw.w.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements the http.Flusher interface
func (tw *timeoutWriter) Flush() {
	if flusher, ok := tw.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// CloseNotify implements the http.CloseNotifier interface
func (tw *timeoutWriter) CloseNotify() <-chan bool {
	if closeNotifier, ok := tw.w.(http.CloseNotifier); ok {
		return closeNotifier.CloseNotify()
	}
	return nil
}

// Push implements the http.Pusher interface
func (tw *timeoutWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := tw.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}
