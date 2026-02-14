package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		expectedLevel  string
		expectedMethod string
		expectedPath   string
	}{
		{
			name:           "successful GET request",
			method:         "GET",
			path:           "/",
			statusCode:     http.StatusOK,
			expectedLevel:  "info",
			expectedMethod: "GET",
			expectedPath:   "/",
		},
		{
			name:           "client error (4xx)",
			method:         "GET",
			path:           "/not-found",
			statusCode:     http.StatusNotFound,
			expectedLevel:  "warn",
			expectedMethod: "GET",
			expectedPath:   "/not-found",
		},
		{
			name:           "server error (5xx)",
			method:         "POST",
			path:           "/api/error",
			statusCode:     http.StatusInternalServerError,
			expectedLevel:  "error",
			expectedMethod: "POST",
			expectedPath:   "/api/error",
		},
		{
			name:           "POST request",
			method:         "POST",
			path:           "/api/data",
			statusCode:     http.StatusCreated,
			expectedLevel:  "info",
			expectedMethod: "POST",
			expectedPath:   "/api/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var logOutput bytes.Buffer
			log := zerolog.New(&logOutput).With().Timestamp().Logger()

			// Create test handler that returns the specified status code
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("test response"))
			})

			// Wrap with logging middleware
			handler := loggingMiddleware(log)(testHandler)

			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("User-Agent", "test-agent")
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, rec.Code)
			}

			// Verify log output contains expected fields
			logStr := logOutput.String()

			if !bytes.Contains(logOutput.Bytes(), []byte(tt.expectedLevel)) {
				t.Errorf("Expected log level %q in output: %s", tt.expectedLevel, logStr)
			}

			if !bytes.Contains(logOutput.Bytes(), []byte(tt.expectedMethod)) {
				t.Errorf("Expected method %q in log output: %s", tt.expectedMethod, logStr)
			}

			if !bytes.Contains(logOutput.Bytes(), []byte(tt.expectedPath)) {
				t.Errorf("Expected path %q in log output: %s", tt.expectedPath, logStr)
			}

			if !bytes.Contains(logOutput.Bytes(), []byte("http request")) {
				t.Errorf("Expected 'http request' message in log output: %s", logStr)
			}

			if !bytes.Contains(logOutput.Bytes(), []byte("duration_ms")) {
				t.Errorf("Expected 'duration_ms' field in log output: %s", logStr)
			}

			if !bytes.Contains(logOutput.Bytes(), []byte("test-agent")) {
				t.Errorf("Expected 'test-agent' user agent in log output: %s", logStr)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusNotFound)

		if rw.statusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rw.statusCode)
		}
	})

	t.Run("captures bytes written", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

		data := []byte("test data")
		n, err := rw.Write(data)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
		}

		if rw.bytesWritten != len(data) {
			t.Errorf("Expected bytesWritten to be %d, got %d", len(data), rw.bytesWritten)
		}
	})

	t.Run("defaults to 200 OK", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

		rw.Write([]byte("test"))

		// If WriteHeader is not called, status should remain 200
		if rw.statusCode != http.StatusOK {
			t.Errorf("Expected default status code %d, got %d", http.StatusOK, rw.statusCode)
		}
	})
}
