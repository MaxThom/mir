package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/maxthom/mir/interfaces/cli"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/rs/zerolog"
)

const (
	AppName = "mir-cli"
	Version = "0.0.1"
)

var (
	cfg = cli.CliConfig{
		LogLevel: "info",
		Target:   "nats://127.0.0.1:4222",
	}
)

func main() {
	ctx := context.Background()
	// Config
	errCfg, lookupFiles, foundFiles := mir_config.New(AppName,
		mir_config.WithEtcFilePath("mir/cli.yaml", mir_config.Yaml, false),
		mir_config.WithXdgConfigHomeFilePath("mir/cli.yaml", mir_config.Yaml, true),
		mir_config.WithEnvVars("MIR"),
	).LoadAndUnmarshal(&cfg)

	// Logger
	// TODO rework this to put kong in the cmd package
	flagDebug := false
	flagLvl := mir_log.LogLevelInfo
	for i, arg := range os.Args[1:] {
		if arg == "--debug" {
			flagDebug = true
		} else if arg == "--log-level" || arg == "-l" {
			if len(os.Args) > i+2 {
				flagLvl = os.Args[i+2]
			}
		}
	}

	var file *os.File
	log := mir_log.Setup(
		mir_log.WithFlagAndFileLogLevel(flagDebug, flagLvl, &cfg.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithXdgConfigHomeLogFile("mir/cli.log", file),
		mir_log.WithAppName(AppName),
	)
	defer file.Close()

	// Finish
	if errCfg != nil {
		log.Err(errCfg).Msg("error loading config")
		os.Exit(1)
	}

	log.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")
	prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(cfg)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling config")
		os.Exit(1)
	}
	log.Info().Str("config", string(prettyCfg)).Msg("")

	// Run!!!
	if err := run(ctx, log, cfg); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	log zerolog.Logger,
	cfg cli.CliConfig,
) error {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	// UI
	ui := cli.NewUI(log, Version, cfg)

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		if err := ui.Launch(ctx); err != nil {
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
