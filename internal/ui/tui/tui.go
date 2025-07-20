package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type Config struct {
	LogLevel string
	Target   string
}

func NewConfig(logLevel string, mirUrl string) Config {
	return Config{Target: mirUrl, LogLevel: logLevel}
}

type Cmd struct{}

func (c *Cmd) Run(log zerolog.Logger, m *mir.Mir, cfg Config) error {
	ctx := context.Background()
	log = log.With().Str("module", "tui").Logger()

	p := tea.NewProgram(NewModel(ctx, l, m, cfg))
	if _, err := p.Run(); err != nil {
		return err
	}

	if err := m.Disconnect(); err != nil {
		log.Error().Err(err).Msg("error disconnecting from Mir server")
	}
	log.Info().Msg("disconnected from Mir server")
	return nil
}
