package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/maxthom/mir/internal/libs/api/health"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/api/pprof"
	"github.com/maxthom/mir/internal/libs/boiler/mir_cli"
	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/internal/libs/build_meta"
	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	AppName = "cockpit"
)

type (
	CockpitConfig struct {
		LogLevel       string
		HttpServer     HttpServer
		CurrentContext string       `yaml:"currentContext"`
		Contexts       []ui.Context `yaml:"contexts"`
	}

	HttpServer struct {
		Port           int
		AllowedOrigins []string `yaml:"allowedOrigins"` // CORS allowed origins (empty = allow all)
	}
)

var (
	defaultCfg = CockpitConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3021,
			AllowedOrigins: []string{
				"http://localhost:5173", // Svelte dev server
				"http://localhost:3021", // Self
			},
		},
		CurrentContext: "local",
		Contexts: []ui.Context{
			{
				Name:    "local",
				Target:  "nats://localhost:4222",
				Grafana: "localhost:3000",
			},
		},
	}
)

func main() {
	ctx := context.Background()

	// cli
	var flagDebug bool
	var flagFilePath string
	var flagLogLevel string

	mir_cli.Setup(AppName,
		mir_cli.WithDescription("Web UI for Mir IoT Hub - Cockpit Dashboard"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithDefaultConfig(&defaultCfg, mir_config.Yaml),
		mir_cli.WithVersion(build_meta.GetShortVersion()),
		mir_cli.WithManual(
			"Serves the Mir Cockpit web interface for device management and monitoring.",
			&defaultCfg, true, ""),
	)
	mir_cli.Parse()

	// File config
	cfg := defaultCfg
	err, lookupFiles, foundFiles := mir_config.
		New(AppName,
			mir_config.WithEtcFilePath("mir/cockpit.yaml", mir_config.Yaml, false),
			mir_config.WithXdgConfigHomeFilePath("mir/cockpit.yaml", mir_config.Yaml, true),
			mir_config.WithFilePath(flagFilePath, mir_config.Yaml, false),
			mir_config.WithEnvVars("mir"),
		).
		LoadAndUnmarshal(&cfg)

	// Log
	log := mir_log.Setup(
		mir_log.WithDevOnlyPrettyLogger(),
		mir_log.WithFlagAndFileLogLevel(flagDebug, flagLogLevel, &cfg.LogLevel),
		mir_log.WithAppName(AppName),
		mir_log.WithTimeFormatUnix(),
	)

	// Finalize and print config
	if err != nil {
		log.Err(err).Msg("")
		os.Exit(1)
	}
	log.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")

	prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling config")
		os.Exit(1)
	}
	log.Info().Str("config", string(prettyCfg)).Msg("")

	// Meta metrics
	metrics.RegisterMirMetrics(AppName, build_meta.GetShortVersion(), map[string]string{}, string(prettyCfg))

	// Run!!!
	if err := run(ctx, log, cfg); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	log zerolog.Logger,
	cfg CockpitConfig,
) error {
	ctx, cancel := mir_signals.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health & Metrics endpoints (these take precedence)
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)
	pprof.RegisterRoutesIfEnvGoPprofSet(mux)

	// Get embedded web filesystem
	webFS, err := fs.Sub(ui.CockpitBuildFS, "web/build")
	if err != nil {
		return fmt.Errorf("failed to get web filesystem: %w", err)
	}

	// SPA handler that serves static files and falls back to index.html
	spaHandler := createSPAHandler(webFS, log)
	mux.Handle("/", spaHandler)

	// Apply middleware stack (order matters: outermost -> innermost)
	// 1. Metrics (tracks all requests including middleware overhead)
	// 2. Logging (logs after metrics tracking starts)
	// 3. Security headers (applies to all responses)
	// 4. CORS (handles cross-origin requests)
	// 5. h2c (HTTP/2 support)
	// 6. Handler (actual request processing)
	handler := metricsMiddleware(mux)
	handler = loggingMiddleware(log)(handler)
	handler = securityHeadersMiddleware(handler)
	handler = corsMiddleware(cfg.HttpServer.AllowedOrigins)(handler)

	// WebServer with HTTP/2 support
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Info().Int("port", cfg.HttpServer.Port).Msg("starting cockpit web server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
			health.SetUnready()
			mir_signals.Shutdown()
		}
		log.Debug().Msg("http server shutdown")
		wg.Done()
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized - navigate to http://localhost:%d", AppName, cfg.HttpServer.Port))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown server")
		}
		shutdownCancel()
		wg.Wait()
		log.Info().Msg("all system have shutdown gracefully")
	})

	return nil
}

// createSPAHandler creates a handler for serving a SPA
// It serves static files if they exist, otherwise falls back to index.html
func createSPAHandler(webFS fs.FS, log zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		requestPath := path.Clean(r.URL.Path)

		// Security: prevent directory traversal
		if strings.Contains(requestPath, "..") {
			log.Warn().Str("path", requestPath).Msg("attempted directory traversal")
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Try to open the file
		filePath := strings.TrimPrefix(requestPath, "/")
		file, err := webFS.Open(filePath)
		if err == nil {
			// File exists, check if it's a directory
			stat, err := file.Stat()
			file.Close()
			if err == nil && !stat.IsDir() {
				// It's a file, serve it with proper caching headers
				w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year for assets
				http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
				return
			}
		}

		// File doesn't exist or is a directory, serve index.html for SPA routing
		indexData, err := fs.ReadFile(webFS, "index.html")
		if err != nil {
			log.Error().Err(err).Msg("failed to read index.html")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // Don't cache index.html
		w.WriteHeader(http.StatusOK)
		w.Write(indexData)
	})
}
