package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

func TestNewProduct(t *testing.T) {
	tests := []struct {
		name      string
		inName    string
		wantField string
	}{
		{"valid", "  Laptop  ", ""},
		{"empty name", "   ", "name"},
		{"name too long", strings.Repeat("a", 101), "name"},
		{"name at limit", strings.Repeat("ñ", 100), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProduct("id-1", tt.inName, " desc ", now)
			assertValidation(t, err, tt.wantField)
			if tt.wantField == "" {
				if p.Name != strings.TrimSpace(tt.inName) || p.Description != "desc" {
					t.Fatalf("fields not trimmed: %+v", p)
				}
				if p.Details == nil || !p.CreatedAt.Equal(now) || !p.UpdatedAt.Equal(now) {
					t.Fatalf("unexpected defaults: %+v", p)
				}
			}
		})
	}
}

func TestNewProductDetail(t *testing.T) {
	tests := []struct {
		name      string
		sku       string
		price     float64
		stock     int
		wantField string
	}{
		{"valid", " SKU-1 ", 10.5, 3, ""},
		{"zero values allowed", "SKU-2", 0, 0, ""},
		{"empty sku", " ", 1, 1, "sku"},
		{"sku too long", strings.Repeat("x", 65), 1, 1, "sku"},
		{"negative price", "SKU-3", -0.01, 1, "price"},
		{"negative stock", "SKU-4", 1, -1, "stock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewProductDetail("d-1", "p-1", tt.sku, tt.price, tt.stock)
			assertValidation(t, err, tt.wantField)
			if tt.wantField == "" && d.SKU != strings.TrimSpace(tt.sku) {
				t.Fatalf("sku not trimmed: %q", d.SKU)
			}
		})
	}
}

func TestFindDetail(t *testing.T) {
	p := Product{Details: []ProductDetail{{ID: "a"}, {ID: "b"}}}
	if got := p.FindDetail("b"); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := p.FindDetail("zzz"); got != -1 {
		t.Fatalf("got %d, want -1", got)
	}
}

func assertValidation(t *testing.T, err error, wantField string) {
	t.Helper()
	if wantField == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("want ValidationError on %q, got %v", wantField, err)
	}
	if verr.Field != wantField {
		t.Fatalf("want field %q, got %q", wantField, verr.Field)
	}
	if verr.Error() == "" {
		t.Fatal("empty error message")
	}
}
