package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var styles = map[string]lipgloss.Style{
	"selectedOption": lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
	"description":    lipgloss.NewStyle().Foreground(lipgloss.Color("#605F63")),
}

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

		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.choices) {
				m.cursor = 0
			}
			return m, nil

		case "up", "k":
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
	s := strings.Builder{}
	if isSelected {
		s.WriteString(styles["selectedOption"].Render("  > "))
		s.WriteString(styles["selectedOption"].Render(o.Label))
	} else {
		s.WriteString("    ")
		s.WriteString(o.Label)
	}
	s.WriteString("\n    ")
	s.WriteString(styles["description"].Render(o.Description))
	s.WriteString("\n\n")
	return s.String()
}

func (m Model) optionSelectedCmd(v OptionValue) tea.Cmd {
	return func() tea.Msg {
		return OptionSelectedMsg{
			v,
		}
	}
}
