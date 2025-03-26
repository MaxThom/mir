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

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/health"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/boiler/mir_cli"
	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/internal/libs/build_meta"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/servers/protocfg_srv"
	"github.com/maxthom/mir/internal/services/schema_cache"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	AppName = "protocfg"
)

type (
	CoreConfig struct {
		LogLevel       string
		HttpServer     HttpServer
		DataBusServer  DataBusServer
		DatabaseServer DatabaseSever
	}

	HttpServer struct {
		Port int
	}

	DataBusServer struct {
		Url string
	}

	DatabaseSever struct {
		Url      string
		User     string
		Password string `cfg:"secret"`
	}
)

var (
	defaultCfg = CoreConfig{
		LogLevel: "info",
		HttpServer: HttpServer{
			Port: 3019,
		},
		DataBusServer: DataBusServer{
			Url: "nats://127.0.0.1:4222",
		},
		DatabaseServer: DatabaseSever{
			Url:      "ws://127.0.0.1:8000/rpc",
			User:     "root",
			Password: "root",
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
		mir_cli.WithDescription("Manage device configuration for Mir"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithDefaultConfig(&defaultCfg, mir_config.Yaml),
		mir_cli.WithVersion(build_meta.GetShortVersion()),
		mir_cli.WithManual(
			"Manage device configuration for Mir with a flow of desired and reported properties",
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
			mir_config.WithEtcFilePath("mir/protocfg.yaml", mir_config.Yaml, false),
			mir_config.WithXdgConfigHomeFilePath("mir/protocfg.yaml", mir_config.Yaml, true),
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
	_ context.Context,
	log zerolog.Logger,
	cfg CoreConfig,
) error {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// Setup
	// Database
	db, err := surreal.ConnectToDb(cfg.DatabaseServer.Url, "global", "mir", cfg.DatabaseServer.User, cfg.DatabaseServer.Password)
	if err != nil {
		return err
	}
	log.Info().Str("url", cfg.DatabaseServer.Url).Str("namespace", "global").Str("database", "mir").Msg("connected to database")

	// Bus
	m, err := mir.Connect("protocfg", cfg.DataBusServer.Url)
	if err != nil {
		return err
	}
	log.Info().Str("url", cfg.DataBusServer.Url).Msg("connected to msg bus")

	// Services
	cc, err := schema_cache.NewMirProtoCache(log, m)
	if err != nil {
		return err
	}
	coreSrv, err := protocfg_srv.NewProtoCfg(log, m, mng.NewSurrealDeviceStore(db), cc)
	if err != nil {
		return err
	}

	// Metrics & Health
	mux := http.NewServeMux()
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)

	// WebServer
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HttpServer.Port),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
			health.SetUnready()
			mir_signals.Shutdown()
		}
		log.Debug().Msg("http server shutdown")
		wg.Done()
	}()

	if err := coreSrv.Serve(); err != nil {
		return err
	}

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown server")
		}
		if err := coreSrv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown core server")
		}
		if err := m.Disconnect(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown Mir")
		}
		db.Close()
		log.Debug().Msg("db conn shutdown")
		shutdownCancel()
		wg.Wait()
		log.Info().Msg("all system have shutdown gracefully")
	})

	return nil
}
