package cli

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/services/cli/root"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type Config struct{}

type TUI struct {
	mirUrl string
}

func NewServer(log zerolog.Logger, mirUrl string) *TUI {
	l = log.With().Str("srv", "cli").Logger()
	return &TUI{mirUrl: mirUrl}
}

func (s *TUI) Launch(ctx context.Context) error {
	p := tea.NewProgram(root.NewModel(ctx, l, s.mirUrl))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
