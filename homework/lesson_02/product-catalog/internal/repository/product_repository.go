package repository

import (
	"errors"
	"sync"
)

type Product struct {
	ID   int
	Name string
}

type ProductRepository interface {
	GetProduct(id int) (*Product, error)
}

type InMemoryProductRepository struct {
	products map[int]Product
	mu       sync.RWMutex
}

func NewInMemoryProductRepository() *InMemoryProductRepository {
	return &InMemoryProductRepository{
		products: map[int]Product{
			1: {ID: 1, Name: "Product A"},
			2: {ID: 2, Name: "Product B"},
			3: {ID: 3, Name: "Product C"},
		},
	}
}

func (r *InMemoryProductRepository) GetProduct(id int) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}

	return &product, nil
}
