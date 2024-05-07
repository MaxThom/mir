package label_checkbox

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
	isChecked       bool
	textinput.Model
	err       error
	validator form.ValidateFn
}

type ()

// IDEA design alternative similar to the textInputs example
// > <label as placeholder> ?
func New(label string, tooltip string, value bool, fn form.ValidateFn) Model {
	i := textinput.New()
	i.CharLimit = 1
	i.Width = 1
	i.Prompt = ""
	i.Cursor.Style = store.Styles["cursor_underline"]
	// TODO find way too blink, top set cursor under the o and remove cursor style
	// IDEA cursor to no style and last letter got the underline
	i.SetCursor(0)
	if value {
		i.SetValue("◉")
	} else {
		i.SetValue("○")
	}

	return Model{
		label:           label,
		tooltip:         tooltip,
		isTooltipHidden: true,
		Model:           i,
		isChecked:       value,
		err:             nil,
		validator:       fn,
	}
}

func (m Model) Init() tea.Cmd {
	return m.Cursor.BlinkCmd()
}

func (m Model) Update(msg tea.Msg) (form.Control, tea.Cmd) {
	cmds := []tea.Cmd{}

	fn := func() {
		if m.Focused() {
			m.isChecked = !m.isChecked
			if m.isChecked {
				m.SetValue("◉")
			} else {
				m.SetValue("○")
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "?":
			m.isTooltipHidden = !m.isTooltipHidden
		case msg.Type == tea.KeyEnter:
			fn()
		case msg.Type == tea.KeySpace:
			fn()
		}
	}
	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	cmds = append(cmds, cmd)
	return &m, tea.Batch(cmds...)
}

func (m Model) View() string {
	// IDEA create a reusable string buffer in the package which is already
	// allocated. This way we can avoid the allocation of the strings.Builder.
	var sb strings.Builder
	sb.WriteString(store.Styles["help"].Render(m.label))
	sb.WriteString(" ")
	if m.tooltip != "" {
		sb.WriteString(store.Styles["help"].Render("? "))
	}
	if m.Focused() {
		sb.WriteString(store.Styles["primary"].Render("> "))
		//		m.Model.TextStyle = store.Styles["primary"]

	} else {
		sb.WriteString(lipgloss.NewStyle().Render("> "))
		//		m.Model.TextStyle = lipgloss.NewStyle()
	}

	sb.WriteString(m.Model.View())
	sb.WriteString(" ")

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
