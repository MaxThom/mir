package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/internal/ui/tui/styles"
)

var v strings.Builder

type (
	OptionSelectedMsg struct {
		Option OptionValue
	}
)
type Model struct {
	cursor  int
	choice  OptionValue
	choices []Option
}
type OptionValue = string
type Option struct {
	Value       OptionValue
	Label       string
	Description string
}

func New(choices []Option) Model {
	return Model{
		choices: choices,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			m.choice = m.choices[m.cursor].Value
			return m, m.optionSelectedCmd(m.choice)

		case "down", "j", "tab":
			m.cursor++
			if m.cursor >= len(m.choices) {
				m.cursor = 0
			}
			return m, nil

		case "up", "k", "shift+tab":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.choices) - 1
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) View() string {
	s := strings.Builder{}
	s.WriteString("\n")

	for i := 0; i < len(m.choices); i++ {
		if m.cursor == i {
			s.WriteString(m.choices[i].String(true))
		} else {
			s.WriteString(m.choices[i].String(false))
		}
	}

	return s.String()
}

func (o Option) String(isSelected bool) string {
	v.Reset()
	if isSelected {
		v.WriteString(styles.Selection.Render("  > "))
		v.WriteString(styles.Selection.Render(o.Label))
	} else {
		v.WriteString("    ")
		v.WriteString(o.Label)
	}
	v.WriteString("\n    ")
	v.WriteString(styles.Subtext.Render(o.Description))
	v.WriteString("\n")
	return v.String()
}

func (m Model) optionSelectedCmd(v OptionValue) tea.Cmd {
	return func() tea.Msg {
		return OptionSelectedMsg{
			v,
		}
	}
}
