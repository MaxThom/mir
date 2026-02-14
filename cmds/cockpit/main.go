package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
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
	cockpit_srv "github.com/maxthom/mir/internal/servers/cockpit_srv"
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

	// Setup HTTP mux
	mux := http.NewServeMux()

	// Register health & metrics endpoints
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)
	pprof.RegisterRoutesIfEnvGoPprofSet(mux)

	// Get embedded web filesystem
	webFS, err := fs.Sub(ui.CockpitBuildFS, "web/build")
	if err != nil {
		return fmt.Errorf("failed to get web filesystem: %w", err)
	}

	// Create cockpit server and register routes
	cockpitSrv, err := cockpit_srv.NewCockpit(log, &cockpit_srv.Options{
		AllowedOrigins: cfg.HttpServer.AllowedOrigins,
		WebFS:          webFS,
	})
	if err != nil {
		return err
	}

	// Register cockpit routes on the mux
	cockpitSrv.RegisterRoutes(mux)

	// Create HTTP server with HTTP/2 support
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	// Start HTTP server in background
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Int("port", cfg.HttpServer.Port).Msg("starting cockpit web server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("http server error")
			health.SetUnready()
			mir_signals.Shutdown()
		}
		log.Debug().Msg("http server shutdown")
	}()

	log.Info().Msg(fmt.Sprintf("%s initialized - navigate to http://localhost:%d", AppName, cfg.HttpServer.Port))
	health.SetReady()

	// Wait for shutdown signal
	mir_signals.WaitForOsSignals(func() {
		cancel()
		log.Info().Msg("shutting down cockpit server")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown server")
		}

		wg.Wait()
		log.Info().Msg("cockpit server shutdown complete")
	})

	return nil
}
