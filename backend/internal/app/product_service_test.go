package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/andresm98/new-project-01/backend/internal/adapters/memory"
	"github.com/andresm98/new-project-01/backend/internal/app"
	"github.com/andresm98/new-project-01/backend/internal/domain"
	"github.com/andresm98/new-project-01/backend/internal/ports"
)

func newService() *app.ProductService {
	n := 0
	ids := func() string { n++; return fmt.Sprintf("id-%d", n) }
	t0 := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { t0 = t0.Add(time.Second); return t0 }
	return app.NewProductService(memory.NewRepository(), ids, clock)
}

func TestProductCRUD(t *testing.T) {
	ctx := context.Background()
	svc := newService()

	p, err := svc.Create(ctx, ports.ProductInput{Name: "Laptop", Description: "14in"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, ports.ProductInput{Name: ""}); !isValidation(err) {
		t.Fatalf("want validation error, got %v", err)
	}

	got, err := svc.Get(ctx, p.ID)
	if err != nil || got.Name != "Laptop" {
		t.Fatalf("get: %v %+v", err, got)
	}

	updated, err := svc.Update(ctx, p.ID, ports.ProductInput{Name: "Laptop Pro"})
	if err != nil || updated.Name != "Laptop Pro" || !updated.UpdatedAt.After(p.UpdatedAt) {
		t.Fatalf("update: %v %+v", err, updated)
	}
	if _, err := svc.Update(ctx, p.ID, ports.ProductInput{Name: " "}); !isValidation(err) {
		t.Fatalf("want validation error, got %v", err)
	}
	if _, err := svc.Update(ctx, "missing", ports.ProductInput{Name: "x"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}

	list, _ := svc.List(ctx)
	if len(list) != 1 {
		t.Fatalf("want 1 product, got %d", len(list))
	}

	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, p.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	if _, err := svc.Get(ctx, p.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestDetails(t *testing.T) {
	ctx := context.Background()
	svc := newService()
	p1, _ := svc.Create(ctx, ports.ProductInput{Name: "Shirt"})
	p2, _ := svc.Create(ctx, ports.ProductInput{Name: "Shoes"})

	d1, err := svc.AddDetail(ctx, p1.ID, ports.DetailInput{SKU: "SH-S", Price: 10, Stock: 5})
	if err != nil {
		t.Fatal(err)
	}
	d2, err := svc.AddDetail(ctx, p1.ID, ports.DetailInput{SKU: "SH-M", Price: 11, Stock: 2})
	if err != nil {
		t.Fatal(err)
	}

	// SKU is unique across all products, case-insensitively.
	if _, err := svc.AddDetail(ctx, p2.ID, ports.DetailInput{SKU: "sh-s", Price: 1}); !errors.Is(err, domain.ErrDuplicateSKU) {
		t.Fatalf("want duplicate sku, got %v", err)
	}
	if _, err := svc.AddDetail(ctx, p1.ID, ports.DetailInput{SKU: "X", Price: -1}); !isValidation(err) {
		t.Fatalf("want validation error, got %v", err)
	}
	if _, err := svc.AddDetail(ctx, "missing", ports.DetailInput{SKU: "X"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}

	// Updating a detail keeping its own SKU is allowed; taking another's is not.
	if _, err := svc.UpdateDetail(ctx, p1.ID, d1.ID, ports.DetailInput{SKU: "SH-S", Price: 12, Stock: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateDetail(ctx, p1.ID, d1.ID, ports.DetailInput{SKU: "SH-M"}); !errors.Is(err, domain.ErrDuplicateSKU) {
		t.Fatalf("want duplicate sku, got %v", err)
	}
	if _, err := svc.UpdateDetail(ctx, p1.ID, d1.ID, ports.DetailInput{SKU: ""}); !isValidation(err) {
		t.Fatalf("want validation error, got %v", err)
	}
	if _, err := svc.UpdateDetail(ctx, p1.ID, "missing", ports.DetailInput{SKU: "Z"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	if _, err := svc.UpdateDetail(ctx, "missing", d1.ID, ports.DetailInput{SKU: "Z"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}

	got, _ := svc.Get(ctx, p1.ID)
	if len(got.Details) != 2 || got.Details[0].Price != 12 {
		t.Fatalf("unexpected details: %+v", got.Details)
	}

	if err := svc.RemoveDetail(ctx, p1.ID, d2.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveDetail(ctx, p1.ID, d2.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	if err := svc.RemoveDetail(ctx, "missing", d2.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}

	// Deleting a product cascades: its SKUs become available again.
	if err := svc.Delete(ctx, p1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddDetail(ctx, p2.ID, ports.DetailInput{SKU: "SH-S"}); err != nil {
		t.Fatalf("sku should be free after cascade delete: %v", err)
	}
}

func isValidation(err error) bool {
	var verr *domain.ValidationError
	return errors.As(err, &verr)
}
