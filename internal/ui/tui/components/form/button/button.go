package button

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/interfaces/tui/components/form"
	"github.com/maxthom/mir/interfaces/tui/styles"
)

var (
	v strings.Builder
)

type Model struct {
	label      string
	isFocused  bool
	prefix     string
	suffix     string
	focusStyle lipgloss.Style
	style      lipgloss.Style
	tooltip    string
	err        error
	validator  form.ValidateFn
}

type (
	ButtonPressedMsg struct {
		Label string
	}
)

func New(label string, tooltip string, fn form.ValidateFn) Model {
	return Model{
		label:      label,
		isFocused:  false,
		prefix:     "[ ",
		suffix:     " ]",
		focusStyle: styles.Primary,
		style:      lipgloss.NewStyle(),
		tooltip:    tooltip,
		err:        nil,
		validator:  fn,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (form.Control, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyEnter:
			if m.Focused() {
				cmds = append(cmds, m.ButtonPressCmd())
			}
		case msg.Type == tea.KeySpace:
			if m.Focused() {
				cmds = append(cmds, m.ButtonPressCmd())
			}
		default:
		}
	}
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	v.Reset()

	if m.Focused() {
		v.WriteString(m.focusStyle.Render(m.prefix))
		v.WriteString(m.style.Render(m.label))
		v.WriteString(m.focusStyle.Render(m.suffix))
	} else {
		v.WriteString(m.style.Render(m.prefix))
		v.WriteString(m.style.Render(m.label))
		v.WriteString(m.style.Render(m.suffix))
	}

	return v.String()
}

func (m *Model) Blur() {
	m.isFocused = false
}

func (m *Model) Focus() tea.Cmd {
	m.isFocused = true
	return nil
}

func (m Model) Focused() bool {
	return m.isFocused
}

func (m Model) GetLabel() string {
	return m.label
}

func (m Model) GetTooltip() string {
	return m.tooltip
}

func (m Model) GetErr() error {
	return m.err
}

func (m Model) GetValue() string {
	return m.label
}

func (m Model) ButtonPressCmd() tea.Cmd {
	return func() tea.Msg {
		return ButtonPressedMsg{m.label}
	}
}

func ButtonPressCmd(s string) tea.Cmd {
	return func() tea.Msg {
		return ButtonPressedMsg{s}
	}
}
