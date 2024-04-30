package cli

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/rs/zerolog"
)

var l zerolog.Logger
var msgBus *bus.BusConn
var ctx context.Context

type (
	CliConfig struct {
		LogLevel string
		Target   string
	}
	Globals struct {
		Target   string      `help:"Mir connection target" default:"nats://127.0.0.1:4222"`
		Debug    bool        `short:"D" help:"Enable debug mode"`
		LogLevel string      `short:"l" help:"Set the logging level (debug|info|warn|error|fatal)" default:"info"`
		Version  VersionFlag `name:"version" help:"Print version information and quit"`
	}
	UI struct {
		Globals

		Device DeviceCmd `cmd:"" help:"Manage fleet of Mir devices"`
	}
	VersionFlag string
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

func NewUI(log zerolog.Logger, version string, cfg CliConfig) *UI {
	l = log.With().Str("srv", "cli").Logger()
	return &UI{
		Globals: Globals{
			Target:  cfg.Target,
			Version: VersionFlag(version),
		},
	}
}

func (u *UI) Launch(c context.Context) error {
	ctx = c
	kongCtx := kong.Parse(u,
		kong.Name("mir"),
		kong.Description("A command line interface to operator the Mir ecosystem 🛰️"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": string(u.Version),
		})
	return kongCtx.Run(&u.Globals)
}

func (u *UI) Run(globals *Globals) error {
	return nil
}
