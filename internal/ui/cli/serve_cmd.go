package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/health"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/boiler/mir_config"
	"github.com/maxthom/mir/internal/libs/boiler/mir_log"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/internal/libs/external/influx"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/servers/core_srv"
	"github.com/maxthom/mir/internal/servers/protocmd_srv"
	"github.com/maxthom/mir/internal/servers/prototlm_srv"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"
)

type (
	ServeConfig struct {
		Mir     MirCfg     `embed:"" prefix:"mir." yaml:"mir"`
		Surreal SurrealCfg `embed:"" prefix:"surreal." yaml:"surreal"`
		Influx  InfluxCfg  `embed:"" prefix:"influx." yaml:"influx"`
	}

	MirCfg struct {
		Url      string `help:"Mir server URL" default:"nats://127.0.0.1:4222" yaml:"url"`
		LogLevel string `help:"Mir loglevel for each service" default:"info" yaml:"logLevel"`
		HttpPort int    `help:"Mir http port for api" default:"3015" yaml:"httpPort"`
	}

	SurrealCfg struct {
		Url       string `help:"Surreal db connection url" default:"ws://127.0.0.1:8000/rpc" yaml:"url"`
		Namespace string `help:"Surreal db namespace" default:"global" yaml:"namespace"`
		Database  string `help:"Surreal db database" default:"mir" yaml:"database"`
		User      string `help:"Surreal db user" default:"root" yaml:"user"`
		Password  string `help:"Surreal db password" default:"root" cfg:"secret" yaml:"password"`
	}

	InfluxCfg struct {
		Url    string `help:"Influx db connection url" default:"http://localhost:8086/" yaml:"url"`
		Token  string `help:"Influx db token" default:"mir-operator-token" cfg:"secret" yaml:"token"`
		Org    string `help:"Influx db organisation" default:"Mir" yaml:"org"`
		Bucket string `help:"Influx db telemetry bucket" default:"mir" yaml:"bucket"`
	}
)

const (
	AppName = "mir"
	Version = "0.3.0"
)

type ServeCmd struct {
	Core              bool   `help:"Core module is for device mangement" default:"true"`
	Telemetry         bool   `help:"Telemetry module is for data ingestion and visualization" default:"true"`
	Command           bool   `help:"Command module if for command and control" default:"true"`
	DisplayDefaultCfg bool   `name:"display-default-cfg" help:"Display default configuration. Can be piped to config file '> ~/.config/mir/mir.yaml'" default:"false"`
	DisplayLoadedCfg  bool   `name:"display-cfg" help:"Display loaded configuration. Usefull for debug scenario" default:"false"`
	ConfigFile        string `name:"server-config" help:"Set path for config path." default:"~/.config/mir/mir.yaml"`

	ServeConfig `embed:"" prefix:""`
}

func (d *ServeCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *ServeCmd) Run(c CLI) error {
	if d.DisplayDefaultCfg {
		out, err := yaml.Marshal(d.ServeConfig)
		if err != nil {
			return fmt.Errorf("error unmarshaling default config: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	err, lookupFiles, foundFiles := mir_config.
		New(AppName,
			mir_config.WithEtcFilePath("mir/mir.yaml", mir_config.Yaml, false),
			mir_config.WithXdgConfigHomeFilePath("mir/mir.yaml", mir_config.Yaml, true),
			mir_config.WithFilePath(d.ConfigFile, mir_config.Yaml, false),
			mir_config.WithEnvVars("mir"),
		).
		LoadAndUnmarshal(&d.ServeConfig)

	if d.DisplayLoadedCfg {
		out, err := yaml.Marshal(d.ServeConfig)
		if err != nil {
			return fmt.Errorf("error unmarshaling loaded config: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	log := mir_log.Setup(
		mir_log.WithDevOnlyPrettyLogger(),
		mir_log.WithFlagAndFileLogLevel(false, d.Mir.LogLevel, &d.Mir.LogLevel),
		mir_log.WithAppName(AppName),
		mir_log.WithTimeFormatUnix(),
	)

	if err != nil {
		log.Err(err).Msg("")
		os.Exit(1)
	}
	log.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")
	prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(d.ServeConfig)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling config")
		os.Exit(1)
	}
	log.Info().Str("config", string(prettyCfg)).Msg("")
	metrics.RegisterMirMetrics(AppName, Version, map[string]string{}, string(prettyCfg))

	if err := run(context.Background(), log, d.ServeConfig); err != nil {
		log.Error().Err(err).Msg("")
		return err
	}
	return nil
}

func run(
	ctx context.Context,
	log zerolog.Logger,
	cfg ServeConfig,
) error {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// Setup
	db, err := surreal.ConnectToDb(cfg.Surreal.Url, cfg.Surreal.Namespace, cfg.Surreal.Database, cfg.Surreal.User, cfg.Surreal.Password)
	if err != nil {
		return err
	}
	log.Info().Str("url", cfg.Surreal.Url).Str("namespace", cfg.Surreal.Namespace).Str("database", cfg.Surreal.Database).Msg("connected to database")

	lpClient := influxdb2.NewClient(cfg.Influx.Url, cfg.Influx.Token)
	if err := influx.CreateOrgAndBucket(ctx, lpClient, cfg.Influx.Org, cfg.Influx.Bucket); err != nil {
		return err
	}
	log.Info().Str("url", cfg.Influx.Url).Msg("connected to puthost")

	m, err := mir.Connect(AppName, cfg.Mir.Url)
	if err != nil {
		return err
	}
	log.Info().Str("url", cfg.Mir.Url).Msg("connected to msg bus")

	// Services
	coreSrv, err := core_srv.NewCore(log, m, mng.NewSurrealDeviceStore(db))
	if err != nil {
		return err
	}
	core_srv.RegisterMetrics(metrics.Registry())

	cmdSrv, err := protocmd_srv.NewProtoCmd(log, m, mng.NewSurrealDeviceStore(db))
	if err != nil {
		return err
	}
	protocmd_srv.RegisterMetrics(metrics.Registry())

	tlmSrv, err := prototlm_srv.NewProtoTlm(log, m, mng.NewSurrealDeviceStore(db), ts.NewInfluxTelemetryStore(cfg.Influx.Org, cfg.Influx.Bucket, lpClient))
	if err != nil {
		return err
	}
	prototlm_srv.RegisterMetrics(metrics.Registry())

	// Metrics & Health
	mux := http.NewServeMux()
	metrics.RegisterRoutes(mux)
	health.RegisterRoutes(mux)

	// WebServer
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Mir.HttpPort),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	log.Info().Msgf("serve on :%d", cfg.Mir.HttpPort)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("")
			health.SetUneady()
			mir_signals.Shutdown()
		}
		log.Debug().Msg("http server shutdown")
		wg.Done()
	}()

	if err := coreSrv.Serve(); err != nil {
		return err
	}
	if err := cmdSrv.Serve(); err != nil {
		return err
	}
	if err := tlmSrv.Serve(); err != nil {
		return err
	}

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	health.SetReady()
	mir_signals.WaitForOsSignals(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatal().Err(err).Msg("failed to gracefully shutdown server")
		}
		if err := coreSrv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown core server")
		}
		if err := cmdSrv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown cmd server")
		}
		if err := tlmSrv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown tlm server")
		}
		m.Disconnect()
		db.Close()
		lpClient.Close()
		log.Debug().Msg("external connections closed")
		shutdownCancel()
		wg.Wait()
		log.Info().Msg("all system have shutdown gracefully")
	})

	return nil
}
