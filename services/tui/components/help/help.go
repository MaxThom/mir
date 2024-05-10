package helpless

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/services/tui/store"
)

type globalKeyMap struct {
	Previous key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = globalKeyMap{
	Previous: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "previous"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
var v strings.Builder

type Model struct {
	tooltips   []string
	keyMap     help.KeyMap
	globalKeys globalKeyMap
	help       help.Model
	cli        string
}

func New(km help.KeyMap, tooltips []string, cli string) Model {
	return Model{
		tooltips:   tooltips,
		keyMap:     km,
		globalKeys: keys,
		help:       help.New(),
		cli:        cli,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.globalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	return m, nil
}

func (m Model) View() string {
	v.Reset()
	if m.help.ShowAll {
		for _, s := range m.tooltips {
			v.WriteString(store.Styles["help"].Render(s))
			v.WriteString("\n")
		}
	}
	v.WriteString("\n")
	v.WriteString(m.help.View(m))
	if m.cli != "" && m.help.ShowAll {
		v.WriteString(store.Styles["help"].Italic(true).Render("\n\n"))
		v.WriteString(store.Styles["help"].Render(m.cli))
	}
	return v.String()
}

func (m *Model) UpdateCli(s string) {
	m.cli = s
}

func (m Model) ShortHelp() []key.Binding {
	return append(m.keyMap.ShortHelp(), m.globalKeys.Help)
}

func (m Model) FullHelp() [][]key.Binding {
	return append(m.keyMap.FullHelp(), []key.Binding{m.globalKeys.Previous, m.globalKeys.Help, m.globalKeys.Quit})
}
