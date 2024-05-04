package label_checkbox

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/services/tui/components/form"
	"github.com/maxthom/mir/services/tui/store"
)

type Model struct {
	label           string
	tooltip         string
	isTooltipHidden bool
	isChecked       bool
	textinput.Model
}

type ()

func New(label string, tooltip string) Model {
	i := textinput.New()
	i.CharLimit = 1
	i.Prompt = ""
	i.SetValue("○")

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
		case msg.Type == tea.KeyEnter:
		case msg.Type == tea.KeySpace:
			if m.Focused() {
				m.isChecked = !m.isChecked
				if m.isChecked {
					m.SetValue("◉")
				} else {
					m.SetValue("○")
				}
			}
		}
	}
	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	cmds = append(cmds, cmd)
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	v := m.label
	t := ""
	if m.tooltip != "" {
		if m.isTooltipHidden {
			t += store.Styles["help"].Render("?")
		} else {
			t += "" + store.Styles["help"].Render(m.tooltip)
		}
	}
	return v + " " + m.Model.View() + " " + t
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
