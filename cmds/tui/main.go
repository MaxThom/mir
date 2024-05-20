package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/maxthom/mir/interfaces/tui"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/rs/zerolog/log"
)

const (
	AppName = "mir-tui"
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
	logFile   *os.File
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

	// Services
	tuiSrv := tui.NewServer(log.Logger, cfg.MirServer)

	go func() {
		if err := tuiSrv.Launch(ctx); err != nil {
			log.Err(err).Msg("")
		}
		mir_signals.Shutdown()
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		cancel()
		go func() {
			logFile.Close()
		}()
	})
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
	errCfg, report := appConfig.LoadAndUnmarshal(&cfg)

	// Logger
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Err(err).Msg("$HOME is not defined")
		userHomeDir = "./"
	}
	dirPath := filepath.Join(userHomeDir, ".config", "mir")
	logPath := filepath.Join(dirPath, "cli.log")
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		log.Err(err).Msg("error creating directories")
		os.Exit(1)
	}

	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Err(err).Msg("can't create file for storing logs")
		os.Exit(1)
	}

	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagDebug {
		cfg.LogLevel = "debug"
	}
	appConfig.Set("logLevel", cfg.LogLevel)
	mir_log.Setup(
		mir_log.WithLogLevel(cfg.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithCustomWriter(logFile),
		mir_log.WithAppName(AppName),
	)

	// Metrics
	metrics.RegisterMirMetrics(AppName, Version, map[string]string{"ca": "dev"}, fmt.Sprintf("%v", appConfig.All()))

	// Finish
	if errCfg != nil {
		log.Err(errCfg).Msg("error loading config")
		os.Exit(1)
	}
	if report != "" {
		log.Warn().Msg(report)
	}

	log.Info().Msg(fmt.Sprintf("%s initializing...", AppName))
	log.Info().Str("mir_config", fmt.Sprintf("%v", appConfig.All())).Msg("mir_config loaded")
}
