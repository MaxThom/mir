package main

import (
	"context"
	"flag"
	"fmt"
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
	"github.com/maxthom/mir/internal/servers/mcp_srv"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

const (
	AppName = "mcp"
)

type (
	CoreConfig struct {
		LogLevel      string
		HttpServer    HttpServer
		DataBusServer DataBusServer
	}

	HttpServer struct {
		Port int
	}

	DataBusServer struct {
		Url string
	}
)

var (
	defaultCfg = CoreConfig{
		LogLevel: "debug",
		HttpServer: HttpServer{
			Port: 3021,
		},
		DataBusServer: DataBusServer{
			Url: "nats://127.0.0.1:4222",
		},
	}
)

func main() {
	ctx := context.Background()

	// cli
	var flagMirTarget string
	var flagDebug bool
	var flagFilePath string
	var flagLogLevel string

	mir_cli.Setup(AppName,
		mir_cli.WithDescription("MCP Server to connect Mir ecosystem to LLM"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithDefaultConfig(&defaultCfg, mir_config.Yaml),
		mir_cli.WithVersion(build_meta.GetShortVersion()),
		mir_cli.WithManual(
			"MCP Server to provides documentation as resources and device management as tools",
			&defaultCfg, true, ""),
		mir_cli.WithOsFlag(func() {
			flag.StringVar(&flagMirTarget, "target", "", "set Mir server url. Default to nats://127.0.0.1:4222.")
		}),
	)
	mir_cli.Parse()

	// File config
	cfg := defaultCfg
	err, lookupFiles, foundFiles := mir_config.
		New(AppName,
			mir_config.WithEtcFilePath("mir/mcp.yaml", mir_config.Yaml, false),
			mir_config.WithXdgConfigHomeFilePath("mir/mcp.yaml", mir_config.Yaml, true),
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

	if flagMirTarget != "" {
		cfg.DataBusServer.Url = flagMirTarget
	}

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
	cfg CoreConfig,
) error {
	ctx, cancel := mir_signals.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT)

	// Setup
	// Bus
	m, err := mir.Connect(AppName, cfg.DataBusServer.Url, append(mir.WithDefaultReconnectOpts(), mir.WithDefaultConnectionLogging(log)...)...)
	if err != nil {
		return err
	}
	log.Info().Str("url", cfg.DataBusServer.Url).Str("status", m.Bus.Status().String()).Msg("msg bus status")

	// Services
	mcpSrv, err := mcp_srv.NewMCP(log, m)
	if err != nil {
		return err
	}

	// Metrics & Health
	mux := http.NewServeMux()
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)
	pprof.RegisterRoutesIfEnvGoPprofSet(mux)

	// WebServer
	// server := &http.Server{
	// 	Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
	// 	Handler: h2c.NewHandler(mux, &http2.Server{}),
	// }

	wg := &sync.WaitGroup{}
	// wg.Add(1)
	// go func() {
	// 	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 		log.Err(err).Msg("")
	// 		health.SetUnready()
	// 		mir_signals.Shutdown()
	// 	}
	// 	log.Debug().Msg("http server shutdown")
	// 	wg.Done()
	// }()

	wg.Add(1)
	mcpHttpSrv := mcpSrv.GetHttpServer()
	go func() {
		if err := mcpHttpSrv.Start(fmt.Sprintf(":%d", cfg.HttpServer.Port)); err != nil {
			log.Err(err).Msg("")
			health.SetUnready()
			mir_signals.Shutdown()
		}

		// if err := mcpSrv.Serve(fmt.Sprintf(":%d", cfg.HttpServer.Port)); err != nil {
		// 	log.Err(err).Msg("")
		// 	health.SetUnready()
		// 	mir_signals.Shutdown()
		// }
		log.Debug().Msg("mcp server shutdown")
		wg.Done()
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		// if err := server.Shutdown(shutdownCtx); err != nil {
		// 	log.Error().Err(err).Msg("failed to gracefully shutdown server")
		// }
		if err := mcpHttpSrv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown core server")
		}
		if err := m.Disconnect(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown Mir")
		}
		shutdownCancel()
		wg.Wait()
		log.Info().Msg("all system have shutdown gracefully")
	})

	return nil
}
