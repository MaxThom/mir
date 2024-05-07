package button

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/services/tui/components/form"
	"github.com/maxthom/mir/services/tui/store"
)

type Model struct {
	label      string
	isFocused  bool
	prefix     string
	suffix     string
	focusStyle lipgloss.Style
	style      lipgloss.Style
}

type (
	ButtonPressedMsg struct {
		Label string
	}
)

func New(label string) Model {
	return Model{
		label:      label,
		isFocused:  false,
		prefix:     "[ ",
		suffix:     " ]",
		focusStyle: store.Styles["primary"],
		style:      lipgloss.NewStyle(),
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
				cmds = append(cmds, m.ButtonPressedCmd())
			}
		case msg.Type == tea.KeySpace:
			if m.Focused() {
				cmds = append(cmds, m.ButtonPressedCmd())
			}
		default:
		}
	}
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var sb strings.Builder

	if m.Focused() {
		sb.WriteString(m.focusStyle.Render(m.prefix))
		sb.WriteString(m.style.Render(m.label))
		sb.WriteString(m.focusStyle.Render(m.suffix))
	} else {
		sb.WriteString(m.style.Render(m.prefix))
		sb.WriteString(m.style.Render(m.label))
		sb.WriteString(m.style.Render(m.suffix))
	}

	return sb.String()
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

func (m Model) ButtonPressedCmd() tea.Cmd {
	return func() tea.Msg {
		return ButtonPressedMsg{m.label}
	}
}
