// Package ports declares the interfaces between the application core and the
// outside world: input ports (driving) and output ports (driven).
package ports

import (
	"context"
	"time"

	"github.com/andresm98/new-project-01/backend/internal/domain"
)

// ProductService is the input port used by driving adapters (HTTP).
type ProductService interface {
	List(ctx context.Context) ([]domain.Product, error)
	Get(ctx context.Context, id string) (domain.Product, error)
	Create(ctx context.Context, in ProductInput) (domain.Product, error)
	Update(ctx context.Context, id string, in ProductInput) (domain.Product, error)
	Delete(ctx context.Context, id string) error
	AddDetail(ctx context.Context, productID string, in DetailInput) (domain.ProductDetail, error)
	UpdateDetail(ctx context.Context, productID, detailID string, in DetailInput) (domain.ProductDetail, error)
	RemoveDetail(ctx context.Context, productID, detailID string) error
}

// ProductInput carries the editable fields of a product.
type ProductInput struct {
	Name        string
	Description string
}

// DetailInput carries the editable fields of a product detail.
type DetailInput struct {
	SKU   string
	Price float64
	Stock int
}

// ProductRepository is the output port for persistence.
type ProductRepository interface {
	List(ctx context.Context) ([]domain.Product, error)
	Get(ctx context.Context, id string) (domain.Product, error)
	Save(ctx context.Context, p domain.Product) error
	Delete(ctx context.Context, id string) error
	// SKUExists reports whether sku is used by any detail other than excludeDetailID.
	SKUExists(ctx context.Context, sku, excludeDetailID string) (bool, error)
}

// IDGenerator is the output port for identity generation.
type IDGenerator func() string

// Clock is the output port for time.
type Clock func() time.Time
