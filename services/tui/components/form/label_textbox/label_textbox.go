package label_textbox

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
	textinput.Model
}

type ()

func New(label string, tooltip string) Model {
	i := textinput.New()

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
	v := m.label
	if m.tooltip != "" {
		if m.isTooltipHidden {
			v += store.Styles["help"].Render(" ?")
		} else {
			v += " " + store.Styles["help"].Render(m.tooltip)
		}
	}
	return v + "\n" + m.Model.View()
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
