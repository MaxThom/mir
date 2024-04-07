package helpless

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

type Model struct {
	keyMap     help.KeyMap
	globalKeys globalKeyMap
	help       help.Model
}

func New(km help.KeyMap) Model {
	return Model{
		keyMap:     km,
		globalKeys: keys,
		help:       help.New(),
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
	return m.help.View(m)
}

func (m Model) ShortHelp() []key.Binding {
	return append(m.keyMap.ShortHelp(), m.globalKeys.Help, m.globalKeys.Previous, m.globalKeys.Quit)
}

func (m Model) FullHelp() [][]key.Binding {
	return append(m.keyMap.FullHelp(), []key.Binding{m.globalKeys.Previous, m.globalKeys.Help, m.globalKeys.Quit})
}
