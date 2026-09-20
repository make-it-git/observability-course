package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"product-catalog/internal/repository"
)

type ProductHandler struct {
	repo   repository.ProductRepository
	logger *log.Logger
}

func NewProductHandler(repo repository.ProductRepository, logger *log.Logger) *ProductHandler {
	return &ProductHandler{
		repo:   repo,
		logger: logger,
	}
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := h.repo.GetProduct(id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	scenario := r.URL.Query().Get("scenario")
	shouldFail := scenario == "" || scenario == "failing"
	// Simulate a random error due to memory issues for example
	// 0.5% chance
	if shouldFail && rand.Intn(200) == 0 {
		allocateRandomMemory(1024 * 1500)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Simulate another random error with timeout
	// 0.1% chance
	if shouldFail && rand.Intn(1000) == 0 {
		n := rand.Intn(3000)
		time.Sleep(time.Duration(n) * time.Millisecond)
		http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(w, `{"id": %d, "name": "%s"}`, product.ID, product.Name)
	if err != nil {
		h.logger.Printf("Error writing response: %v", err)
		return
	}
}

func allocateRandomMemory(size int) []byte {
	data := make([]byte, size)

	rand.Seed(time.Now().UnixNano())
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	return data
}
