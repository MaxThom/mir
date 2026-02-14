package cockpit_srv

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

type CockpitServer struct {
	log  zerolog.Logger
	opts Options
}

type Options struct {
	AllowedOrigins []string
	WebFS          fs.FS     // Embedded web files
	Config         ui.Config // Cockpit configuration
}

func NewCockpit(logger zerolog.Logger, opts *Options) (*CockpitServer, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.WebFS == nil {
		return nil, fmt.Errorf("webFS cannot be nil")
	}

	return &CockpitServer{
		log:  logger.With().Str("srv", "cockpit_server").Logger(),
		opts: *opts,
	}, nil
}

// RegisterRoutes registers cockpit HTTP handlers on the provided mux
// The handlers are wrapped with middleware (metrics, logging, security, CORS)
func (s *CockpitServer) RegisterRoutes(mux *http.ServeMux) {
	// API endpoint - config endpoint with middleware
	// Must be registered BEFORE the catch-all "/" route
	apiHandler := metricsMiddleware(http.HandlerFunc(s.configHandler))
	apiHandler = loggingMiddleware(s.log)(apiHandler)
	apiHandler = securityHeadersMiddleware(apiHandler)
	apiHandler = corsMiddleware(s.opts.AllowedOrigins)(apiHandler)
	mux.Handle("/api/contexts", apiHandler)

	// Create SPA handler that serves static files and falls back to index.html
	spaHandler := createSPAHandler(s.opts.WebFS, s.log)

	// Apply middleware stack (order matters: outermost -> innermost)
	// 1. Metrics (tracks all requests including middleware overhead)
	// 2. Logging (logs after metrics tracking starts)
	// 3. Security headers (applies to all responses)
	// 4. CORS (handles cross-origin requests)
	// 5. Handler (actual request processing)
	handler := metricsMiddleware(spaHandler)
	handler = loggingMiddleware(s.log)(handler)
	handler = securityHeadersMiddleware(handler)
	handler = corsMiddleware(s.opts.AllowedOrigins)(handler)

	// Register the wrapped handler for all cockpit routes
	// This serves the SPA and all its static assets
	mux.Handle("/", handler)

	s.log.Debug().Msg("cockpit routes registered")
}
