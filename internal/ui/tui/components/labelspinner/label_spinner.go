package labelspinner

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
)

var v strings.Builder

type Model struct {
	lbl         string
	spinner     spinner.Model
	IsSpinning  bool
	prefix      string
	placeholder string
	timer       timer.Model
}

type (
	StartMsg struct{}
	StopMsg  struct{}
)

func New(prefix string, placeholder string, spinIcon spinner.Spinner) Model {
	s := spinner.New()
	s.Spinner = spinIcon
	return Model{
		prefix:      prefix,
		placeholder: placeholder,
		spinner:     s,
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

func (m *Model) UpdateLabelWithTimeout(label string, timeout time.Duration) tea.Cmd {
	m.lbl = label
	m.timer = timer.New(timeout)
	return m.timer.Init()
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
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		m.lbl = m.placeholder
	}
	return m, nil
}

func (m Model) View() string {
	v.Reset()
	if m.IsSpinning {
		v.WriteString(m.prefix + " " + m.spinner.View() + m.lbl)
	} else {
		v.WriteString(m.prefix + " " + m.lbl)
	}
	return v.String()
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
