package label_textbox

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/services/tui/components/form"
	"github.com/maxthom/mir/services/tui/store"
)

var (
	v strings.Builder
)

type Model struct {
	label           string
	tooltip         string
	isTooltipHidden bool
	textinput.Model
	err       error
	validator form.ValidateFn
}

func New(label string, tooltip string, fn form.ValidateFn) Model {
	i := textinput.New()
	i.Prompt = store.Styles["primary"].Render("> ")

	return Model{
		label:           label,
		tooltip:         tooltip,
		isTooltipHidden: true,
		Model:           i,
		err:             nil,
		validator:       fn,
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

			if m.validator != nil {
				m.err = m.validator(m.Value())
			}
		}
	}
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	v.Reset()
	if m.tooltip != "" {
		v.WriteString(store.Styles["help"].Render("? "))
	} else {
		v.WriteString("  ")
	}
	v.WriteString(store.Styles["form_label"].Bold(false).Render(m.label))
	v.WriteString(" ")
	if m.Focused() {
		m.Model.Prompt = store.Styles["primary"].Render("> ")
	} else {
		m.Model.Prompt = "> "
	}
	v.WriteString(m.Model.View())
	v.WriteString(" ")

	return v.String()
}

func (m *Model) SetValue(s string) {
	m.Model.SetValue(s)
	if m.validator != nil {
		m.err = m.validator(m.Value())
	}
}

func (m Model) GetValue() string {
	return m.Value()
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

func (m Model) Focused() bool {
	return m.Model.Focused()
}

func (m Model) GetLabel() string {
	return m.label
}

func (m Model) GetErr() error {
	return m.err
}

func (m Model) GetTooltip() string {
	return m.tooltip
}
