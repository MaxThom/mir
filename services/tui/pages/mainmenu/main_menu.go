package mainmenu

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/components/menu"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/rs/zerolog/log"
)

var (
	l                                     = log.With().Str("cmp", "mainmenu").Logger()
	menuOption_devices   menu.OptionValue = "/devices"
	menuOption_twins     menu.OptionValue = "/twins"
	menuOption_telemetry menu.OptionValue = "/telemetry"
)

type Model struct {
	menu         menu.Model
	currentRoute menu.OptionValue
	help         mir_help.Model
}

var styles = map[string]lipgloss.Style{
	"mir": lipgloss.NewStyle().Foreground(lipgloss.Color("#C26BFF")),
}

func NewModel() Model {
	mm := menu.New([]menu.Option{
		{
			Value:       menuOption_devices,
			Label:       "devices",
			Description: "create and manage a fleet of Mir devices",
		},
		{
			Value:       menuOption_twins,
			Label:       "(wip) twins",
			Description: "manage a set of default twins template for devices configuration",
		},
		{
			Value:       menuOption_telemetry,
			Label:       "(wip) telemetry",
			Description: "upload new telemetry schema",
		},
	})
	return Model{
		menu: mm,
		help: mir_help.New(keys),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case menu.OptionSelectedMsg:
		m.currentRoute = msg.Option
		return m, msgs.RouteChangeCmd(msg.Option)
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		default:
			m.menu, cmd = m.menu.Update(msg)
			if cmd != nil {
				return m, cmd
			}
			m.help, cmd = m.help.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	var v strings.Builder
	v.WriteString(m.menu.View())
	v.WriteString("\n\n" + m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["up"], k["down"], k["enter"]},
	}
}

var keys = keyMap{
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	"enter": key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "select"),
	),
}
