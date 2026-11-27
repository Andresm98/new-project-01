// Package memory is an in-memory implementation of ports.ProductRepository.
package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/andresm98/new-project-01/backend/internal/domain"
	"github.com/andresm98/new-project-01/backend/internal/ports"
)

// Repository stores products in a map guarded by a RWMutex.
type Repository struct {
	mu       sync.RWMutex
	products map[string]domain.Product
}

var _ ports.ProductRepository = (*Repository)(nil)

// NewRepository returns an empty repository.
func NewRepository() *Repository {
	return &Repository{products: map[string]domain.Product{}}
}

func (r *Repository) List(_ context.Context) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, clone(p))
	}
	slices.SortFunc(out, func(a, b domain.Product) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (r *Repository) Get(_ context.Context, id string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return clone(p), nil
}

func (r *Repository) Save(_ context.Context, p domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = clone(p)
	return nil
}

func (r *Repository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.products, id) // details are removed with their product (cascade)
	return nil
}

func (r *Repository) SKUExists(_ context.Context, sku, excludeDetailID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.products {
		for _, d := range p.Details {
			if d.ID != excludeDetailID && strings.EqualFold(d.SKU, sku) {
				return true, nil
			}
		}
	}
	return false, nil
}

// clone copies the details slice so callers never share memory with the store.
func clone(p domain.Product) domain.Product {
	p.Details = append([]domain.ProductDetail{}, p.Details...)
	return p
}
