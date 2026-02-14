package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsMiddleware(t *testing.T) {
	// Reset metrics before test
	httpRequestsTotal.Reset()
	httpRequestDuration.Reset()
	httpRequestSize.Reset()
	httpResponseSize.Reset()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	handler := metricsMiddleware(testHandler)

	// Make a request (use / since sanitizePath groups non-API paths)
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify request counter incremented (sanitizePath("/dashboard") = "/")
	count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/", "200"))
	if count != 1 {
		t.Errorf("Expected request count to be 1, got %f", count)
	}

	// Verify response was successful
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMetricsMiddleware_InFlightGauge(t *testing.T) {
	// Reset gauge
	httpRequestsInFlight.Set(0)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that in-flight gauge is incremented during request
		inFlight := testutil.ToFloat64(httpRequestsInFlight)
		if inFlight != 1 {
			t.Errorf("Expected in-flight requests to be 1, got %f", inFlight)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := metricsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// After request completes, gauge should be back to 0
	inFlight := testutil.ToFloat64(httpRequestsInFlight)
	if inFlight != 0 {
		t.Errorf("Expected in-flight requests to be 0 after request, got %f", inFlight)
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/metrics", "/metrics"},
		{"/health", "/health"},
		{"/api/devices", "/api/devices"},
		{"/", "/"},
		{"/dashboard", "/"},
		{"/devices/123", "/"},
		{"/static/app.js", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeApproximateRequestSize(t *testing.T) {
	tests := []struct {
		name        string
		setupReq    func() *http.Request
		expectMin   int
		description string
	}{
		{
			name: "simple GET request",
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expectMin:   10,
			description: "Should calculate basic request size",
		},
		{
			name: "request with headers",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("User-Agent", "test")
				req.Header.Set("Accept", "application/json")
				return req
			},
			expectMin:   50,
			description: "Should include header sizes",
		},
		{
			name: "POST request with body",
			setupReq: func() *http.Request {
				body := strings.NewReader("test body content")
				req := httptest.NewRequest("POST", "/api/data", body)
				req.ContentLength = 17
				return req
			},
			expectMin:   30,
			description: "Should include body size from Content-Length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			size := computeApproximateRequestSize(req)

			if size < tt.expectMin {
				t.Errorf("%s: expected size >= %d, got %d", tt.description, tt.expectMin, size)
			}
		})
	}
}

func TestMetricsResponseWriter(t *testing.T) {
	t.Run("captures status code and bytes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mrw := &metricsResponseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

		mrw.WriteHeader(http.StatusCreated)
		data := []byte("response data")
		n, err := mrw.Write(data)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		if mrw.statusCode != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, mrw.statusCode)
		}

		if mrw.bytesWritten != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), mrw.bytesWritten)
		}
	})
}
