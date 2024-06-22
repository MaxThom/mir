package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/maxthom/mir/interfaces/tui"
	"github.com/maxthom/mir/libs/boiler/mir_cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/rs/zerolog"
)

const (
	AppName = "mir-tui"
	Version = "0.0.1"
)

var (
	defaultCfg = CliConfig{
		LogLevel:  "info",
		MirServer: "nats://127.0.0.1:4222",
	}
)

type (
	CliConfig struct {
		LogLevel  string
		MirServer string
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
		mir_cli.WithDescription("Connect and operate the Mir ecosystem"),
		mir_cli.WithConfigFilePath(&flagFilePath),
		mir_cli.WithLogLevel(&flagLogLevel),
		mir_cli.WithLogDebug(&flagDebug),
		mir_cli.WithManual(
			"Interact with the Mir Ecosystem with a lower level and more admin tool to manage your fleet.",
			&defaultCfg, true, ""),
		mir_cli.WithOsFlag(func() {
			flag.StringVar(&flagMirTarget, "target", "", "set Mir server url. Default to nats://127.0.0.1:4222.")
		}),
	)
	mir_cli.Parse()

	// Config
	cfg := defaultCfg
	errCfg, lookupFiles, foundFiles := mir_config.New(AppName,
		mir_config.WithEtcFilePath("config.yaml", mir_config.Yaml, false),
		mir_config.WithXdgConfigHomeFilePath("config.yaml", mir_config.Yaml, true),
		mir_config.WithFilePath(flagFilePath, mir_config.Yaml, false),
		mir_config.WithEnvVars("MIR"),
	).LoadAndUnmarshal(&cfg)

	// Logger
	var file *os.File
	log := mir_log.Setup(
		mir_log.WithFlagAndFileLogLevel(flagDebug, flagLogLevel, &cfg.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithXdgConfigHomeLogFile("mir/cli.log", file),
		mir_log.WithAppName(AppName),
	)
	defer file.Close()

	// Finalize and print config
	log.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")

	if errCfg != nil {
		log.Err(errCfg).Msg("error loading config")
		os.Exit(1)
	}
	if flagMirTarget != "" {
		cfg.MirServer = flagMirTarget
	}

	prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(cfg)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling config")
		os.Exit(1)
	}
	log.Info().Str("config", string(prettyCfg)).Msg("")

	// Run!!!
	if err := run(ctx, log, cfg); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}
}

func run(ctx context.Context, log zerolog.Logger, cfg CliConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// Services
	tuiSrv := tui.New(log, cfg.MirServer)

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		if err := tuiSrv.Launch(ctx); err != nil {
			log.Err(err).Msg("")
		}
		mir_signals.Shutdown()
		wg.Done()
	}()

	// Handle shutdown
	log.Info().Msg(fmt.Sprintf("%s initialized", AppName))
	mir_signals.WaitForOsSignals(func() {
		cancel()
		wg.Wait()
		log.Info().Msg("shutdown")
	})
	return nil
}
