package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/internal/ui"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
)

type Cmd struct{}

func (c *Cmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	ctx := context.Background()
	log = log.With().Str("module", "tui").Logger()

	store.MirCfg = cfg
	mCtx, _ := cfg.GetCurrentContext()
	store.MirCtx = mCtx
	p := tea.NewProgram(NewModel(ctx, log, m, cfg))
	if _, err := p.Run(); err != nil {
		return err
	}

	if err := m.Disconnect(); err != nil {
		log.Error().Err(err).Msg("error disconnecting from Mir server")
	}
	log.Info().Msg("disconnected from Mir server")
	return nil
}
