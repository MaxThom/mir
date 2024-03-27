package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/cli"
	"github.com/nats-io/nats.go"
	logger "github.com/rs/zerolog/log"
)

const (
	AppName = "mir-cli"
	Version = "0.0.1"
)

var (
	flagDebug    bool
	flagFilePath string
	flagLogLevel string

	cfg = CliConfig{
		LogLevel:  "info",
		MirServer: "nats://127.0.0.1:4222",
	}
	appConfig = mir_config.Empty()
	log       = logger.With().Str("cmd", AppName).Logger()
)

type (
	CliConfig struct {
		LogLevel  string
		MirServer string
	}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// Setup
	// Bus
	b, err := createMirConnection(ctx)
	if err != nil {
		log.Err(err).Msg("")
	}
	log.Info().Str("url", cfg.MirServer).Msg("connected to Mir 🛰️")

	// Services
	cliSrv := cli.NewServer(log, b)

	go func() {
		cliSrv.Launch()
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		cancel()
		go func() {
			b.Drain()
			b.Close()
		}()
	})
}

// TODO
// rework this function to a library or something
func createMirConnection(ctx context.Context) (*bus.BusConn, error) {
	b, err := bus.New(cfg.MirServer,
		bus.WithReconnHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("reconnected to " + nc.ConnectedUrl())
		}),
		bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
			logger.Warn().Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			logger.Warn().Msg("connection to %v closed " + nc.ConnectedUrl())
		}))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func init() {
	// Cli
	mir_cli.Setup(AppName,
		mir_cli.WithDescription("Interact with the Mir Ecosystem to manage your fleet of devices"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithManual(
			"Interact with the Mir Ecosystem with a lower level and more admin tool to manage your fleet.",
			&cfg, true, ""),
		mir_cli.WithOsFlag(func() {
		}),
	)
	mir_cli.Parse()

	// Config
	opts := []func(*mir_config.MirConfig){
		mir_config.WithEtcFilePath("config.yaml", mir_config.Yaml, false),
		mir_config.WithXdgConfigHomeFilePath("config.yaml", mir_config.Yaml, true),
		mir_config.WithEnvVars(),
	}
	if flagFilePath != "" {
		opts = append(opts, mir_config.WithFilePath(flagFilePath, mir_config.Yaml, false))
	}
	appConfig = mir_config.New(AppName, opts...)
	err, warns := appConfig.LoadAndUnmarshal(&cfg)

	// Logger
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagDebug {
		cfg.LogLevel = "debug"
	}
	appConfig.Set("logLevel", cfg.LogLevel)
	mir_log.Setup(mir_log.WithLogLevel(cfg.LogLevel), mir_log.WithTimeFormatUnix())

	// Metrics
	metrics.RegisterMirMetrics(AppName, Version, map[string]string{"ca": "dev"}, fmt.Sprintf("%v", appConfig.All()))

	// Finish
	if err != nil {
		log.Err(err).Msg("")
		os.Exit(1)
	}
	if warns != nil {
		log.Warn().Msg(warns.Error())
	}

	log.Info().Msg(fmt.Sprintf("%s initializing...", AppName))
	log.Info().Str("mir_config", fmt.Sprintf("%v", appConfig.All())).Msg("mir_config loaded")
}
