package group_menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/internal/ui/tui/styles"
)

var v strings.Builder

type (
	OptionSelectedMsg struct {
		Option OptionValue
	}
)
type Model struct {
	cursor       int
	choice       *Option
	groupChoices []GroupChoice
	totalChoices int
}

type GroupChoice struct {
	Label   string
	Choices []Option
}

type OptionValue = string
type Option struct {
	Value       OptionValue
	Label       string
	Description string
	Details     string
	IsOpen      bool
}

func New(choices []GroupChoice) Model {
	total := 0
	for _, gc := range choices {
		total += len(gc.Choices)
	}
	var ch *Option
	if len(choices) > 0 && len(choices[0].Choices) > 0 {
		ch = &choices[0].Choices[0]
	}
	return Model{
		choice:       ch,
		groupChoices: choices,
		totalChoices: total,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeySpace.String():
			if m.choice != nil {
				m.choice.IsOpen = !m.choice.IsOpen
				return m, m.optionSelectedCmd(m.choice.Value)
			}
			return m, nil
		case "down", "j", "tab":
			m.cursor++
			if m.cursor >= m.totalChoices {
				m.cursor = 0
			}
			m.choice = findOptionFromCursor(m.cursor, m.groupChoices)
			return m, m.optionSelectedCmd(m.choice.Value)
		case "up", "k", "shift+tab":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = m.totalChoices - 1
			}
			m.choice = findOptionFromCursor(m.cursor, m.groupChoices)
			return m, m.optionSelectedCmd(m.choice.Value)
		}
	}
	return m, nil
}

func findOptionFromCursor(cursor int, groupChoices []GroupChoice) *Option {
	tempCursor := cursor
	for _, gc := range groupChoices {
		if tempCursor < len(gc.Choices) {
			return &gc.Choices[tempCursor]
		}
		tempCursor -= len(gc.Choices)
	}
	return &Option{}
}

func (m Model) View() string {
	s := strings.Builder{}
	iC, jC := m.GetCursor()
	for i, groupCh := range m.groupChoices {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render(strings.TrimSuffix("  "+groupCh.Label+":", " ")))
		s.WriteString("\n")
		for j, ch := range groupCh.Choices {
			if iC == i && jC == j {
				s.WriteString(ch.String(true))
			} else {
				s.WriteString(ch.String(false))
			}
		}
		s.WriteString("\n")
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
	if o.Description != "" {
		v.WriteString("\n")
		v.WriteString(styles.Subtext.Render(o.Description))
	}
	if o.Details != "" && o.IsOpen {
		v.WriteString("\n")
		v.WriteString(styles.Subtext.Render(o.Details))
	}
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

func (m Model) GetCurrentChoice() OptionValue {
	return m.choice.Value
}

func (m Model) GetFlatCursor() int {
	return m.cursor
}

func (m Model) GetCursor() (int, int) {
	tempCursor := m.cursor
	for i, gc := range m.groupChoices {
		if tempCursor < len(gc.Choices) {
			return i, tempCursor
		}
		tempCursor -= len(gc.Choices)
	}
	return 0, tempCursor
}
