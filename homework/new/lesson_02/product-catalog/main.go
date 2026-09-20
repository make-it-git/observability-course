// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	_ "product-catalog/internal/metrics"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"product-catalog/internal/config"
	"product-catalog/internal/handlers"
	"product-catalog/internal/repository"
)

const (
	defaultPort = "8080"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger := log.New(os.Stdout, "product-catalog: ", log.LstdFlags)

	productRepo := repository.NewInMemoryProductRepository()

	productHandler := handlers.NewProductHandler(productRepo, logger)

	router := mux.NewRouter()
	router.Use(handlers.HttpMetricsMiddleware)

	router.HandleFunc("/products/{id}", productHandler.GetProduct).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()) // Expose Prometheus metrics

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "OK")
	}).Methods("GET")

	port := cfg.Port
	if port == "" {
		port = defaultPort
	}
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Printf("Server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on %s: %v\n", addr, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown failed: %v", err)
	}
	logger.Println("Server exited properly")
}
