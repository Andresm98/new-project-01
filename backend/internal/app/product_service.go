// Package app implements the use cases. It depends only on domain and ports.
package app

import (
	"context"
	"sync"

	"github.com/andresm98/new-project-01/backend/internal/domain"
	"github.com/andresm98/new-project-01/backend/internal/ports"
)

// ProductService implements ports.ProductService.
type ProductService struct {
	repo  ports.ProductRepository
	newID ports.IDGenerator
	now   ports.Clock
	// mu serialises writes so the SKU uniqueness check and the save are atomic.
	mu sync.Mutex
}

var _ ports.ProductService = (*ProductService)(nil)

// NewProductService wires the use cases with their output ports.
func NewProductService(repo ports.ProductRepository, newID ports.IDGenerator, now ports.Clock) *ProductService {
	return &ProductService{repo: repo, newID: newID, now: now}
}

func (s *ProductService) List(ctx context.Context) ([]domain.Product, error) {
	return s.repo.List(ctx)
}

func (s *ProductService) Get(ctx context.Context, id string) (domain.Product, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProductService) Create(ctx context.Context, in ports.ProductInput) (domain.Product, error) {
	p, err := domain.NewProduct(s.newID(), in.Name, in.Description, s.now())
	if err != nil {
		return domain.Product{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.repo.Save(ctx, p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ProductService) Update(ctx context.Context, id string, in ports.ProductInput) (domain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	if err := p.Update(in.Name, in.Description, s.now()); err != nil {
		return domain.Product{}, err
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) AddDetail(ctx context.Context, productID string, in ports.DetailInput) (domain.ProductDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.repo.Get(ctx, productID)
	if err != nil {
		return domain.ProductDetail{}, err
	}
	d, err := domain.NewProductDetail(s.newID(), productID, in.SKU, in.Price, in.Stock)
	if err != nil {
		return domain.ProductDetail{}, err
	}
	if err := s.ensureUniqueSKU(ctx, d.SKU, ""); err != nil {
		return domain.ProductDetail{}, err
	}
	p.Details = append(p.Details, d)
	p.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, p); err != nil {
		return domain.ProductDetail{}, err
	}
	return d, nil
}

func (s *ProductService) UpdateDetail(ctx context.Context, productID, detailID string, in ports.DetailInput) (domain.ProductDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.repo.Get(ctx, productID)
	if err != nil {
		return domain.ProductDetail{}, err
	}
	i := p.FindDetail(detailID)
	if i < 0 {
		return domain.ProductDetail{}, domain.ErrNotFound
	}
	d := p.Details[i]
	if err := d.Update(in.SKU, in.Price, in.Stock); err != nil {
		return domain.ProductDetail{}, err
	}
	if err := s.ensureUniqueSKU(ctx, d.SKU, detailID); err != nil {
		return domain.ProductDetail{}, err
	}
	p.Details[i] = d
	p.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, p); err != nil {
		return domain.ProductDetail{}, err
	}
	return d, nil
}

func (s *ProductService) RemoveDetail(ctx context.Context, productID, detailID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.repo.Get(ctx, productID)
	if err != nil {
		return err
	}
	i := p.FindDetail(detailID)
	if i < 0 {
		return domain.ErrNotFound
	}
	p.Details = append(p.Details[:i], p.Details[i+1:]...)
	p.UpdatedAt = s.now()
	return s.repo.Save(ctx, p)
}

func (s *ProductService) ensureUniqueSKU(ctx context.Context, sku, excludeDetailID string) error {
	exists, err := s.repo.SKUExists(ctx, sku, excludeDetailID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrDuplicateSKU
	}
	return nil
}
