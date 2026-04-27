package cockpit_srv

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

type CockpitServer struct {
	log   zerolog.Logger
	opts  Options
	store mng.DashboardStore
	cache *releasesCache
}

type Options struct {
	AllowedOrigins []string
	WebFS          fs.FS              // Embedded web files
	Config         ui.Config          // Cockpit configuration
	Store          mng.DashboardStore // Dashboard persistence (optional)
	GitHub         GitHubOptions
}

type GitHubOptions struct {
	Owner string
	Repo  string
}

func NewCockpit(logger zerolog.Logger, opts *Options) (*CockpitServer, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.WebFS == nil {
		return nil, fmt.Errorf("webFS cannot be nil")
	}

	return &CockpitServer{
		log:   logger.With().Str("srv", "cockpit_server").Logger(),
		opts:  *opts,
		store: opts.Store,
		cache: &releasesCache{},
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
	mux.Handle("/api/v1/contexts", apiHandler)

	// Credentials endpoint - POST with optional password, returns .creds file content
	credsHandler := metricsMiddleware(http.HandlerFunc(s.credentialsHandler))
	credsHandler = loggingMiddleware(s.log)(credsHandler)
	credsHandler = securityHeadersMiddleware(credsHandler)
	credsHandler = corsMiddleware(s.opts.AllowedOrigins)(credsHandler)
	mux.Handle("/api/v1/credentials", credsHandler)

	// GitHub releases endpoint (only if owner is configured)
	if s.opts.GitHub.Owner != "" {
		s.registerDashboardRoute(mux, "/api/v1/releases", s.releasesHandler)
	}

	// Dashboard API routes (only if store is configured)
	if s.store != nil {
		if err := SeedWelcomeDashboard(s.log, s.store); err != nil {
			s.log.Error().Err(err).Msg("failed to seed welcome dashboard")
		}

		s.registerDashboardRoute(mux, "/api/v1/dashboards", s.dashboardsHandler)
		s.registerDashboardRoute(mux, "/api/v1/dashboards/", s.dashboardHandler)
	}

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

func (s *CockpitServer) registerDashboardRoute(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	h := metricsMiddleware(http.HandlerFunc(handler))
	h = loggingMiddleware(s.log)(h)
	h = securityHeadersMiddleware(h)
	h = corsMiddleware(s.opts.AllowedOrigins)(h)
	mux.Handle(pattern, h)
}
