package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/andresm98/new-project-01/backend/internal/adapters/http"
	"github.com/andresm98/new-project-01/backend/internal/adapters/memory"
	"github.com/andresm98/new-project-01/backend/internal/adapters/metrics"
	"github.com/andresm98/new-project-01/backend/internal/app"
)

func newServer(t *testing.T, origin string) *httptest.Server {
	t.Helper()
	n := 0
	svc := app.NewProductService(memory.NewRepository(), func() string { n++; return fmt.Sprintf("id-%d", n) }, time.Now)
	srv := httptest.NewServer(httpadapter.NewRouter(svc, metrics.New(), origin))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, body string) (int, map[string]any, string) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	return resp.StatusCode, obj, string(raw)
}

func errCode(obj map[string]any) string {
	e, _ := obj["error"].(map[string]any)
	s, _ := e["code"].(string)
	return s
}

func TestProductsAPI(t *testing.T) {
	srv := newServer(t, "")

	status, _, raw := do(t, srv, "GET", "/api/v1/products", "")
	if status != 200 || strings.TrimSpace(raw) != "[]" {
		t.Fatalf("empty list: %d %s", status, raw)
	}

	status, p, _ := do(t, srv, "POST", "/api/v1/products", `{"name":"Laptop","description":"14in"}`)
	if status != 201 || p["id"] == "" {
		t.Fatalf("create: %d %v", status, p)
	}
	id := p["id"].(string)

	cases := []struct {
		name, method, path, body string
		wantStatus               int
		wantCode                 string
	}{
		{"get missing", "GET", "/api/v1/products/nope", "", 404, "not_found"},
		{"create empty name", "POST", "/api/v1/products", `{"name":""}`, 400, "validation_error"},
		{"create bad json", "POST", "/api/v1/products", `{`, 400, "invalid_body"},
		{"create unknown field", "POST", "/api/v1/products", `{"name":"x","foo":1}`, 400, "invalid_body"},
		{"update missing", "PUT", "/api/v1/products/nope", `{"name":"x"}`, 404, "not_found"},
		{"update invalid", "PUT", "/api/v1/products/" + id, `{"name":""}`, 400, "validation_error"},
		{"update bad json", "PUT", "/api/v1/products/" + id, `[`, 400, "invalid_body"},
		{"detail negative price", "POST", "/api/v1/products/" + id + "/details", `{"sku":"A","price":-1,"stock":1}`, 400, "validation_error"},
		{"detail bad json", "POST", "/api/v1/products/" + id + "/details", `x`, 400, "invalid_body"},
		{"detail missing product", "POST", "/api/v1/products/nope/details", `{"sku":"A","price":1,"stock":1}`, 404, "not_found"},
		{"update detail missing", "PUT", "/api/v1/products/" + id + "/details/nope", `{"sku":"A","price":1,"stock":1}`, 404, "not_found"},
		{"update detail bad json", "PUT", "/api/v1/products/" + id + "/details/nope", `x`, 400, "invalid_body"},
		{"delete detail missing", "DELETE", "/api/v1/products/" + id + "/details/nope", "", 404, "not_found"},
		{"delete missing", "DELETE", "/api/v1/products/nope", "", 404, "not_found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, obj, raw := do(t, srv, c.method, c.path, c.body)
			if status != c.wantStatus || errCode(obj) != c.wantCode {
				t.Fatalf("got %d %s", status, raw)
			}
		})
	}

	status, d, _ := do(t, srv, "POST", "/api/v1/products/"+id+"/details", `{"sku":"LP-1","price":999.9,"stock":3}`)
	if status != 201 || d["productId"] != id {
		t.Fatalf("add detail: %d %v", status, d)
	}
	detailID := d["id"].(string)

	if status, obj, _ := do(t, srv, "POST", "/api/v1/products/"+id+"/details", `{"sku":"LP-1","price":1,"stock":1}`); status != 409 || errCode(obj) != "duplicate_sku" {
		t.Fatalf("duplicate sku: %d %v", status, obj)
	}

	if status, obj, _ := do(t, srv, "PUT", "/api/v1/products/"+id+"/details/"+detailID, `{"sku":"LP-1","price":899,"stock":2}`); status != 200 || obj["price"] != 899.0 {
		t.Fatalf("update detail: %d %v", status, obj)
	}
	if status, obj, _ := do(t, srv, "PUT", "/api/v1/products/"+id, `{"name":"Laptop Pro"}`); status != 200 || obj["name"] != "Laptop Pro" {
		t.Fatalf("update product: %d %v", status, obj)
	}

	status, got, _ := do(t, srv, "GET", "/api/v1/products/"+id, "")
	if status != 200 || len(got["details"].([]any)) != 1 {
		t.Fatalf("get with details: %d %v", status, got)
	}

	if status, _, _ := do(t, srv, "DELETE", "/api/v1/products/"+id+"/details/"+detailID, ""); status != 204 {
		t.Fatalf("delete detail: %d", status)
	}
	if status, _, _ := do(t, srv, "DELETE", "/api/v1/products/"+id, ""); status != 204 {
		t.Fatalf("delete product: %d", status)
	}
}

func TestHealthAndMetrics(t *testing.T) {
	srv := newServer(t, "")

	if status, obj, _ := do(t, srv, "GET", "/healthz", ""); status != 200 || obj["status"] != "ok" {
		t.Fatalf("healthz: %d %v", status, obj)
	}
	do(t, srv, "GET", "/api/v1/products/abc-123", "")
	do(t, srv, "GET", "/api/v1/products/def-456", "")
	do(t, srv, "GET", "/does-not-exist", "")

	_, _, body := do(t, srv, "GET", "/metrics", "")
	want := []string{
		`http_requests_total{method="GET",route="/api/v1/products/{id}",status="404"} 2`,
		`http_requests_total{method="GET",route="unmatched",status="404"} 1`,
		`http_request_duration_seconds_bucket{method="GET",route="/api/v1/products/{id}",status="404",le="+Inf"} 2`,
		`http_requests_in_flight 0`,
		`go_goroutines`,
	}
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("metrics missing %q", w)
		}
	}
	// Raw IDs must never become label values, and /metrics and /healthz are not instrumented.
	for _, bad := range []string{"abc-123", `route="/metrics"`, `route="/healthz"`} {
		if strings.Contains(body, bad) {
			t.Errorf("metrics must not contain %q", bad)
		}
	}
}

func TestCORS(t *testing.T) {
	srv := newServer(t, "http://localhost:8080")
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/v1/products", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 || resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Fatalf("preflight: %d %v", resp.StatusCode, resp.Header)
	}
	resp, _ = http.Get(srv.URL + "/api/v1/products")
	resp.Body.Close()
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Fatal("missing CORS header on simple request")
	}

	noCors := newServer(t, "")
	resp, _ = http.Get(noCors.URL + "/api/v1/products")
	resp.Body.Close()
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS must be disabled when no origin is configured")
	}
}
