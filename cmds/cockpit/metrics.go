package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cockpit_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cockpit_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"method", "path", "status"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cockpit_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100, 1000, 10000, ..., 100000000
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cockpit_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cockpit_http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)
)

// metricsResponseWriter wraps http.ResponseWriter to capture metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// metricsMiddleware records Prometheus metrics for HTTP requests
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap the response writer
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			bytesWritten:   0,
		}

		// Record request size
		requestSize := computeApproximateRequestSize(r)
		httpRequestSize.WithLabelValues(r.Method, sanitizePath(r.URL.Path)).Observe(float64(requestSize))

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		path := sanitizePath(r.URL.Path)

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, path, status).Observe(float64(wrapped.bytesWritten))
	})
}

// sanitizePath normalizes paths to reduce cardinality in metrics
// This prevents metrics explosion from dynamic path segments
func sanitizePath(path string) string {
	// For now, keep paths as-is
	// In the future, you might want to:
	// - Group all /api/* paths
	// - Replace UUIDs/IDs with placeholders
	// - Limit to specific known paths

	// Skip metrics endpoint to avoid recursion
	if path == "/metrics" {
		return "/metrics"
	}

	// Skip health endpoint
	if path == "/health" {
		return "/health"
	}

	// Group all static assets under one label
	if len(path) > 1 && (path[0] == '/' && path[1] != 'a') {
		// If it's not /api/*, it's probably a static asset or SPA route
		return "/"
	}

	return path
}

// computeApproximateRequestSize computes the approximate size of the HTTP request
func computeApproximateRequestSize(r *http.Request) int {
	size := 0

	// Request line: METHOD PATH PROTOCOL
	size += len(r.Method)
	size += len(r.URL.Path)
	size += len(r.Proto)
	size += 4 // spaces and \r\n

	// Headers
	for name, values := range r.Header {
		size += len(name) + 2 // name + ": "
		for _, value := range values {
			size += len(value) + 2 // value + \r\n
		}
	}
	size += 2 // Final \r\n

	// Body (if Content-Length is set)
	if r.ContentLength > 0 {
		size += int(r.ContentLength)
	}

	return size
}
