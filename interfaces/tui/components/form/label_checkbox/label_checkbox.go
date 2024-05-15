package label_checkbox

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/interfaces/tui/components/form"
	"github.com/maxthom/mir/interfaces/tui/styles"
)

var (
	v strings.Builder
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

// IDEA design alternative similar to the textInputs example
// > <label as placeholder> ?
func New(label string, tooltip string, value bool, fn form.ValidateFn) Model {
	i := textinput.New()
	i.CharLimit = 1
	i.Width = 1
	i.Prompt = ""
	i.Cursor.Style = styles.CursorUnderline
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
	v.Reset()
	if m.tooltip != "" {
		v.WriteString(styles.Help.Render("? "))
	} else {
		v.WriteString("  ")
	}

	v.WriteString(styles.FormLabel.Render(m.label))
	v.WriteString(" ")
	if m.Focused() {
		v.WriteString(styles.Primary.Render("> "))

	} else {
		v.WriteString(lipgloss.NewStyle().Render("> "))
	}

	v.WriteString(m.Model.View())
	v.WriteString(" ")

	return v.String()
}

func (m Model) GetValue() string {
	return strconv.FormatBool(m.isChecked)
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
