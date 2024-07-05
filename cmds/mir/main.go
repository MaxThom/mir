package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/maxthom/mir/interfaces/cli"
	"github.com/maxthom/mir/interfaces/tui"
	"github.com/maxthom/mir/libs/boiler/mir_config"
	"github.com/maxthom/mir/libs/boiler/mir_log"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/rs/zerolog"
)

var l zerolog.Logger
var msgBus *bus.BusConn
var ctx context.Context

type configFlag string

type (
	config struct {
		LogLevel string
		Target   string
	}
	Globals struct {
		Target     string      `help:"Mir connection target. default:nats://127.0.0.1:4222"`
		Debug      bool        `short:"D" help:"Enable debug mode"`
		LogLevel   string      `short:"l" help:"Set the logging level (debug|info|warn|error|fatal). default:info"`
		ConfigFile configFlag  `short:"c" help:"Set path for config path. default:~/.config/mir/mir.yaml"`
		Version    VersionFlag `name:"version" help:"Print version information and quit"`
	}
	CLI struct {
		Globals

		Tui    tui.Cmd       `cmd:"" help:"Open Mir in TUI mode" default:"withargs" hidden:""`
		Device cli.DeviceCmd `cmd:"" help:"Manage fleet of Mir devices"`
	}
	VersionFlag string
)

const (
	AppName = "mir"
	Version = "0.0.1"
)

var (
	defaultCfg = config{
		LogLevel: "info",
		Target:   "nats://127.0.0.1:4222",
	}
)

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println(vars["version"])
	app.Exit(0)
	return nil
}

// TODO
// - [ ] return mir error from client in services/core for Publish

func main() {
	var c CLI

	// CLI
	kongCtx := kong.Parse(&c,
		kong.Name("mir"),
		kong.Description("A command line and terminal user interface to operate the Mir ecosystem 🛰️"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": string(Version),
		})

	// Config
	cfg := defaultCfg
	errCfg, lookupFiles, foundFiles := mir_config.New(AppName,
		mir_config.WithEtcFilePath("mir.yaml", mir_config.Yaml, false),
		mir_config.WithXdgConfigHomeFilePath("mir.yaml", mir_config.Yaml, false),
		mir_config.WithFilePath(string(c.ConfigFile), mir_config.Yaml, false),
		mir_config.WithEnvVars("MIR"),
	).LoadAndUnmarshal(&cfg)

	var file *os.File
	log := mir_log.Setup(
		mir_log.WithFlagAndFileLogLevel(c.Debug, c.LogLevel, &cfg.LogLevel),
		mir_log.WithTimeFormatUnix(),
		mir_log.WithXdgConfigHomeLogFile("mir/mir.log", file),
		mir_log.WithAppName(AppName),
	)
	defer file.Close()

	log.Info().Strs("lookup config", lookupFiles).Strs("found config", foundFiles).Msg("configuration loaded")
	if errCfg != nil {
		log.Err(errCfg).Msg("error loading config")
		os.Exit(1)
	}
	if c.LogLevel != "" {
		cfg.LogLevel = c.LogLevel
	}
	if c.Target != "" {
		cfg.Target = c.Target
	}

	prettyCfg, err := mir_config.JsonMarshalWithoutSecrets(cfg)
	if err != nil {
		log.Error().Err(err).Msg("error marshalling config")
		os.Exit(1)
	}
	log.Info().Str("config", string(prettyCfg)).Msg("")

	err = kongCtx.Run(
		cli.New(log, cfg.Target),
		tui.New(log, cfg.Target),
	)
	if err != nil {
		log.Error().Err(err).Msg("")
		fmt.Println(err)
		os.Exit(1)
	}
}
