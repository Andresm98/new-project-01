// Package http is the driving adapter that exposes the product use cases as a
// JSON REST API.
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/andresm98/new-project-01/backend/internal/domain"
	"github.com/andresm98/new-project-01/backend/internal/ports"
)

const maxBodyBytes = 1 << 20

// Handler translates HTTP requests into calls on the ProductService port.
type Handler struct {
	svc ports.ProductService
}

// NewHandler builds a Handler.
func NewHandler(svc ports.ProductService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the API routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/products", h.listProducts)
	mux.HandleFunc("POST /api/v1/products", h.createProduct)
	mux.HandleFunc("GET /api/v1/products/{id}", h.getProduct)
	mux.HandleFunc("PUT /api/v1/products/{id}", h.updateProduct)
	mux.HandleFunc("DELETE /api/v1/products/{id}", h.deleteProduct)
	mux.HandleFunc("POST /api/v1/products/{id}/details", h.addDetail)
	mux.HandleFunc("PUT /api/v1/products/{id}/details/{detailId}", h.updateDetail)
	mux.HandleFunc("DELETE /api/v1/products/{id}/details/{detailId}", h.removeDetail)
}

// --- DTOs ---------------------------------------------------------------

type productRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type detailRequest struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type detailResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"productId"`
	SKU       string  `json:"sku"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
}

type productResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	Details     []detailResponse `json:"details"`
}

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func toDetailResponse(d domain.ProductDetail) detailResponse {
	return detailResponse{ID: d.ID, ProductID: d.ProductID, SKU: d.SKU, Price: d.Price, Stock: d.Stock}
}

func toProductResponse(p domain.Product) productResponse {
	details := make([]detailResponse, 0, len(p.Details))
	for _, d := range p.Details {
		details = append(details, toDetailResponse(d))
	}
	return productResponse{
		ID: p.ID, Name: p.Name, Description: p.Description,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Details: details,
	}
}

// --- Handlers -----------------------------------------------------------

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]productResponse, 0, len(products))
	for _, p := range products {
		out = append(out, toProductResponse(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(p))
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Create(r.Context(), ports.ProductInput{Name: req.Name, Description: req.Description})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProductResponse(p))
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if !decode(w, r, &req) {
		return
	}
	p, err := h.svc.Update(r.Context(), r.PathValue("id"), ports.ProductInput{Name: req.Name, Description: req.Description})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(p))
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addDetail(w http.ResponseWriter, r *http.Request) {
	var req detailRequest
	if !decode(w, r, &req) {
		return
	}
	d, err := h.svc.AddDetail(r.Context(), r.PathValue("id"), ports.DetailInput{SKU: req.SKU, Price: req.Price, Stock: req.Stock})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDetailResponse(d))
}

func (h *Handler) updateDetail(w http.ResponseWriter, r *http.Request) {
	var req detailRequest
	if !decode(w, r, &req) {
		return
	}
	d, err := h.svc.UpdateDetail(r.Context(), r.PathValue("id"), r.PathValue("detailId"),
		ports.DetailInput{SKU: req.SKU, Price: req.Price, Stock: req.Stock})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetailResponse(d))
}

func (h *Handler) removeDetail(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveDetail(r.Context(), r.PathValue("id"), r.PathValue("detailId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ------------------------------------------------------------

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{errorPayload{"invalid_body", "request body must be valid JSON"}})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	var verr *domain.ValidationError
	switch {
	case errors.As(err, &verr):
		writeJSON(w, http.StatusBadRequest, errorBody{errorPayload{"validation_error", verr.Error()}})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{errorPayload{"not_found", "resource not found"}})
	case errors.Is(err, domain.ErrDuplicateSKU):
		writeJSON(w, http.StatusConflict, errorBody{errorPayload{"duplicate_sku", err.Error()}})
	default:
		slog.Error("unexpected error", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorBody{errorPayload{"internal_error", "internal server error"}})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}
