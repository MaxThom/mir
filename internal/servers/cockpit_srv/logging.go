package cockpit_srv

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// loggingMiddleware logs HTTP requests with structured logging
func loggingMiddleware(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code and bytes
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status
				bytesWritten:   0,
			}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log the request
			duration := time.Since(start)

			// Create log event
			logEvent := log.Info()

			// Add status level (error, warn, info) based on status code
			if wrapped.statusCode >= 500 {
				logEvent = log.Error()
			} else if wrapped.statusCode >= 400 {
				logEvent = log.Warn()
			}

			logEvent.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Int("status", wrapped.statusCode).
				Int("bytes", wrapped.bytesWritten).
				Dur("duration_ms", duration).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Str("referer", r.Referer()).
				Msg("http request")
		})
	}
}
