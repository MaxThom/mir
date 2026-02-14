package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with security headers middleware
	handler := securityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify security headers are set
	tests := []struct {
		header   string
		expected bool
	}{
		{"Content-Security-Policy", true},
		{"X-Frame-Options", true},
		{"X-Content-Type-Options", true},
		{"X-XSS-Protection", true},
		{"Referrer-Policy", true},
		{"Permissions-Policy", true},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			value := rec.Header().Get(tt.header)
			if tt.expected && value == "" {
				t.Errorf("Expected header %s to be set, but it was empty", tt.header)
			}
		})
	}

	// Verify specific values
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

func TestCORSMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expectAllowed  bool
	}{
		{
			name:           "empty allowed origins allows all",
			allowedOrigins: []string{},
			requestOrigin:  "http://example.com",
			expectAllowed:  true,
		},
		{
			name:           "specific origin is allowed",
			allowedOrigins: []string{"http://localhost:5173"},
			requestOrigin:  "http://localhost:5173",
			expectAllowed:  true,
		},
		{
			name:           "non-allowed origin is blocked",
			allowedOrigins: []string{"http://localhost:5173"},
			requestOrigin:  "http://evil.com",
			expectAllowed:  false,
		},
		{
			name:           "multiple origins - first is allowed",
			allowedOrigins: []string{"http://localhost:5173", "http://localhost:3020"},
			requestOrigin:  "http://localhost:5173",
			expectAllowed:  true,
		},
		{
			name:           "multiple origins - second is allowed",
			allowedOrigins: []string{"http://localhost:5173", "http://localhost:3020"},
			requestOrigin:  "http://localhost:3020",
			expectAllowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := corsMiddleware(tt.allowedOrigins)(testHandler)

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")

			if tt.expectAllowed {
				if len(tt.allowedOrigins) == 0 && allowOrigin != "*" {
					t.Errorf("Expected Access-Control-Allow-Origin = *, got %q", allowOrigin)
				} else if len(tt.allowedOrigins) > 0 && allowOrigin != tt.requestOrigin {
					t.Errorf("Expected Access-Control-Allow-Origin = %q, got %q", tt.requestOrigin, allowOrigin)
				}
			} else {
				if allowOrigin == tt.requestOrigin {
					t.Errorf("Expected origin %q to be blocked, but it was allowed", tt.requestOrigin)
				}
			}
		})
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	})

	handler := corsMiddleware([]string{"http://localhost:5173"})(testHandler)

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify preflight response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d for preflight request, got %d", http.StatusOK, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Expected Access-Control-Allow-Methods to be set for preflight request")
	}

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Expected Access-Control-Allow-Headers to be set for preflight request")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"item exists", []string{"a", "b", "c"}, "b", true},
		{"item does not exist", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"empty string in slice", []string{"a", "", "c"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.slice, tt.item); got != tt.expected {
				t.Errorf("contains() = %v, want %v", got, tt.expected)
			}
		})
	}
}
