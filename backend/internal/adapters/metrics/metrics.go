// Package metrics exposes the four golden signals as Prometheus metrics.
package metrics

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics groups the HTTP collectors and their registry.
type Metrics struct {
	registry *prometheus.Registry
	duration *prometheus.HistogramVec
	requests *prometheus.CounterVec
	inFlight prometheus.Gauge
}

// New registers the golden-signal collectors plus Go/process runtime metrics
// (saturation) in a dedicated registry.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		// Latency.
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"method", "route", "status"}),
		// Traffic and errors (status=~"5..").
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "route", "status"}),
		// Saturation.
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		}),
	}
	m.registry.MustRegister(
		m.duration, m.requests, m.inFlight,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// Handler serves the /metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware instruments every request except the skipped paths (metrics and
// health checks, so they don't skew the signals).
func (m *Metrics) Middleware(next http.Handler, skip ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(skip, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		elapsed := time.Since(start).Seconds()

		// http.ServeMux sets r.Pattern on the request it routed. Use the
		// pattern, never the raw path, to keep label cardinality bounded.
		labels := prometheus.Labels{
			"method": r.Method,
			"route":  routeLabel(r.Pattern),
			"status": strconv.Itoa(rec.status),
		}
		m.duration.With(labels).Observe(elapsed)
		m.requests.With(labels).Inc()
	})
}

// routeLabel strips the optional "METHOD " prefix from a ServeMux pattern.
func routeLabel(pattern string) string {
	if pattern == "" {
		return "unmatched"
	}
	if i := strings.IndexByte(pattern, ' '); i >= 0 {
		return pattern[i+1:]
	}
	return pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
