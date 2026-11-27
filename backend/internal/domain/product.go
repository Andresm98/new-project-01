// Package domain holds the business entities and rules. It must not depend on
// any other layer or third-party library.
package domain

import (
	"strings"
	"time"
)

const (
	maxNameLength = 100
	maxSKULength  = 64
)

// Product is the master entity.
type Product struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Details     []ProductDetail
}

// ProductDetail is the detail entity (a variant of a product).
type ProductDetail struct {
	ID        string
	ProductID string
	SKU       string
	Price     float64
	Stock     int
}

// NewProduct builds a validated product.
func NewProduct(id, name, description string, now time.Time) (Product, error) {
	p := Product{ID: id, CreatedAt: now, Details: []ProductDetail{}}
	if err := p.Update(name, description, now); err != nil {
		return Product{}, err
	}
	return p, nil
}

// Update changes the editable fields of a product.
func (p *Product) Update(name, description string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return NewValidationError("name", "is required")
	}
	if len([]rune(name)) > maxNameLength {
		return NewValidationError("name", "must be at most 100 characters")
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	p.UpdatedAt = now
	return nil
}

// NewProductDetail builds a validated detail.
func NewProductDetail(id, productID, sku string, price float64, stock int) (ProductDetail, error) {
	d := ProductDetail{ID: id, ProductID: productID}
	if err := d.Update(sku, price, stock); err != nil {
		return ProductDetail{}, err
	}
	return d, nil
}

// Update changes the editable fields of a detail.
func (d *ProductDetail) Update(sku string, price float64, stock int) error {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return NewValidationError("sku", "is required")
	}
	if len(sku) > maxSKULength {
		return NewValidationError("sku", "must be at most 64 characters")
	}
	if price < 0 {
		return NewValidationError("price", "must be greater than or equal to 0")
	}
	if stock < 0 {
		return NewValidationError("stock", "must be greater than or equal to 0")
	}
	d.SKU = sku
	d.Price = price
	d.Stock = stock
	return nil
}

// FindDetail returns the index of a detail, or -1.
func (p *Product) FindDetail(detailID string) int {
	for i, d := range p.Details {
		if d.ID == detailID {
			return i
		}
	}
	return -1
}
