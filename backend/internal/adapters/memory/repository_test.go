package memory

import (
	"context"
	"testing"
	"time"

	"github.com/andresm98/new-project-01/backend/internal/domain"
)

func TestRepositoryIsolatesStoredData(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	p := domain.Product{ID: "p1", Name: "A", Details: []domain.ProductDetail{{ID: "d1", SKU: "S1"}}}
	_ = r.Save(ctx, p)

	// Mutating the caller's copy must not change what is stored.
	p.Details[0].SKU = "CHANGED"
	got, _ := r.Get(ctx, "p1")
	if got.Details[0].SKU != "S1" {
		t.Fatalf("store shares memory with caller: %+v", got.Details)
	}
	got.Details[0].SKU = "CHANGED"
	again, _ := r.Get(ctx, "p1")
	if again.Details[0].SKU != "S1" {
		t.Fatalf("store shares memory with reader: %+v", again.Details)
	}
}

func TestRepositoryListOrderAndSKU(t *testing.T) {
	ctx := context.Background()
	r := NewRepository()
	t0 := time.Now()
	_ = r.Save(ctx, domain.Product{ID: "b", CreatedAt: t0.Add(time.Second)})
	_ = r.Save(ctx, domain.Product{ID: "a", CreatedAt: t0, Details: []domain.ProductDetail{{ID: "d1", SKU: "abc"}}})
	_ = r.Save(ctx, domain.Product{ID: "c", CreatedAt: t0})

	list, _ := r.List(ctx)
	if len(list) != 3 || list[0].ID != "a" || list[1].ID != "c" || list[2].ID != "b" {
		t.Fatalf("unexpected order: %v", ids(list))
	}

	if ok, _ := r.SKUExists(ctx, "ABC", ""); !ok {
		t.Fatal("sku lookup must be case-insensitive")
	}
	if ok, _ := r.SKUExists(ctx, "abc", "d1"); ok {
		t.Fatal("excluded detail must be ignored")
	}
	if err := r.Delete(ctx, "zzz"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func ids(ps []domain.Product) []string {
	out := []string{}
	for _, p := range ps {
		out = append(out, p.ID)
	}
	return out
}
