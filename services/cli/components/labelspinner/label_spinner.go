package labelspinner

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	lbl        string
	spinner    spinner.Model
	IsSpinning bool
	prefix     string
}

type (
	StartMsg struct{}
	StopMsg  struct{}
)

func New(prefix string, spinIcon spinner.Spinner) Model {
	s := spinner.New()
	s.Spinner = spinIcon
	return Model{
		prefix:  prefix,
		spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		return m.spinner.Tick
	}
}

func (m *Model) UpdateLabel(label string) {
	m.lbl = label
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartMsg:
		m.IsSpinning = true
		return m, m.spinner.Tick

	case StopMsg:
		m.IsSpinning = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	return m.prefix + m.spinner.View() + m.lbl
}

func (m Model) Tick() tea.Msg {
	return m.spinner.Tick()
}

func (m Model) Start() tea.Cmd {
	return func() tea.Msg {
		return StartMsg{}
	}
}

func (m Model) Stop() tea.Cmd {
	return func() tea.Msg {
		return StopMsg{}
	}
}
