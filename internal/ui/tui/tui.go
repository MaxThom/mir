package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type TUI struct {
	Target string
}

func New(log zerolog.Logger, mirUrl string) TUI {
	l = log.With().Str("app", "tui").Logger()
	return TUI{Target: mirUrl}
}

type Cmd struct{}

func (c *Cmd) Run(tui TUI) error {
	ctx := context.Background()
	tui.Launch(ctx)
	return nil
}

func (s *TUI) Launch(ctx context.Context) error {
	p := tea.NewProgram(NewModel(ctx, l, s.Target))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
