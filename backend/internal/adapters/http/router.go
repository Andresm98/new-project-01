package http

import (
	"net/http"

	"github.com/andresm98/new-project-01/backend/internal/adapters/metrics"
	"github.com/andresm98/new-project-01/backend/internal/ports"
)

// NewRouter assembles the API, health and metrics endpoints with the
// middleware chain: CORS -> metrics -> mux.
func NewRouter(svc ports.ProductService, m *metrics.Metrics, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()
	NewHandler(svc).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", m.Handler())

	return cors(m.Middleware(mux, "/metrics", "/healthz"), allowedOrigin)
}

// cors enables cross-origin calls only when an origin is configured. With the
// default deployment the frontend proxies /api, so requests are same-origin.
func cors(next http.Handler, allowedOrigin string) http.Handler {
	if allowedOrigin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", allowedOrigin)
		h.Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
