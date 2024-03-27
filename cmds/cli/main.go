package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/services/cli"
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

	// Services
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error().Msg("$HOME is not defined")
	}
	dirPath := filepath.Join(userHomeDir, ".config", "mir")
	logPath := filepath.Join(dirPath, "cli.log")
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		log.Error().Msg("Error creating directories")
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err) // Handle the error according to your needs
	}
	defer logFile.Close()
	log = log.Output(logFile)

	cliSrv := cli.NewServer(log, cfg.MirServer)

	go func() {
		cliSrv.Launch(ctx)
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		cancel()
		go func() {
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
