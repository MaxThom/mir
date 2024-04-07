package devices

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bus "github.com/maxthom/mir/libs/external/natsio"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/rs/zerolog/log"
)

var (
	l = log.With().Str("cmp", "root").Logger()
)

type Model struct {
	ctx  context.Context
	bus  *bus.BusConn
	help mir_help.Model
}

var styles = map[string]lipgloss.Style{
	"mir": lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")),
}

func NewModel(ctx context.Context, bus *bus.BusConn) *Model {
	return &Model{
		ctx:  ctx,
		bus:  bus,
		help: mir_help.New(keys),
	}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		default:
			m.help, cmd = m.help.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	var v strings.Builder
	v.WriteString(m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["create"], k["edit"]},
		{k["up"], k["down"]},
	}
}

var keys = keyMap{
	"create": key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create"),
	),
	"edit": key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
}
