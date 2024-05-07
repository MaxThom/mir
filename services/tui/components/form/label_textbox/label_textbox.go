package label_textbox

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/services/tui/components/form"
	"github.com/maxthom/mir/services/tui/store"
)

type Model struct {
	label           string
	tooltip         string
	isTooltipHidden bool
	textinput.Model
}

type ()

func New(label string, tooltip string) Model {
	i := textinput.New()
	i.Prompt = store.Styles["primary"].Render("> ")

	return Model{
		label:           label,
		tooltip:         tooltip,
		isTooltipHidden: true,
		Model:           i,
	}
}

func (m Model) Init() tea.Cmd {
	return m.Cursor.BlinkCmd()
}

func (m Model) Update(msg tea.Msg) (form.Control, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch {
		case msg.String() == "?":
			m.isTooltipHidden = !m.isTooltipHidden
		default:
			var cmd tea.Cmd
			m.Model, cmd = m.Model.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(store.Styles["help"].Bold(false).Render(m.label))
	sb.WriteString(" ")
	if m.tooltip != "" {
		sb.WriteString(store.Styles["help"].Render("? "))
		if !m.isTooltipHidden {
			sb.WriteString(store.Styles["help"].Render(m.tooltip))
			sb.WriteString(lipgloss.NewStyle().Render(" "))
		}
	}
	sb.WriteString("\n")
	if m.Focused() {
		m.Model.Prompt = store.Styles["primary"].Render("> ")
		//m.Model.TextStyle = store.Styles["primary"]

	} else {
		m.Model.Prompt = "> "
		//m.Model.TextStyle = lipgloss.NewStyle()
	}
	sb.WriteString(m.Model.View())

	return sb.String()
}

func (m Model) Blink() tea.Msg {
	return m.Model.Cursor.BlinkCmd()
}

func (m *Model) Blur() {
	m.Model.Blur()
}

func (m *Model) Focus() tea.Cmd {
	return m.Model.Focus()
}
