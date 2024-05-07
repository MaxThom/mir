package device_create

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/services/tui/components/form"
	"github.com/maxthom/mir/services/tui/components/form/button"
	"github.com/maxthom/mir/services/tui/components/form/label_checkbox"
	"github.com/maxthom/mir/services/tui/components/form/label_textbox"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields
// Use an interface with blur and focus with each form type implement
// Have an array of form types
var (
	l = log.With().Str("page", "device_create").Logger()
)

const (
	deviceId = iota
	name
	description
	disabled
	labels
	annotations
	cancel
	submit
)

type ()

type Model struct {
	ctx     context.Context
	help    mir_help.Model
	inputs  []form.Control
	focused int
}

func NewModel(ctx context.Context) *Model {
	inputs := make([]form.Control, 8)
	tiId := label_textbox.New("Unique ID", "Suggestions are existing IDs")

	tiId.CharLimit = 50
	tiId.Width = 60
	tiId.Validate = deviceIdValidator
	tiId.Focus()
	inputs[deviceId] = &tiId

	tiNm := label_textbox.New("Name", "")
	tiNm.CharLimit = 50
	tiNm.Width = 60
	tiNm.Validate = deviceIdValidator
	inputs[name] = &tiNm

	tiDesc := label_textbox.New("Description", "")
	tiDesc.CharLimit = 50
	tiDesc.Width = 60
	tiDesc.Validate = deviceIdValidator
	inputs[description] = &tiDesc

	tiLbls := label_textbox.New("Labels", "Set of indexed key value pairs to identify the device <k1=v1;k2=v2>")
	tiDesc.CharLimit = 50
	tiDesc.Width = 60
	tiDesc.Validate = deviceIdValidator
	inputs[labels] = &tiLbls

	tiAnno := label_textbox.New("Annotations", "Set of key value pairs to add information on the device <k1=v1;k2=v2>")
	tiDesc.CharLimit = 50
	tiDesc.Width = 60
	tiDesc.Validate = deviceIdValidator
	inputs[annotations] = &tiAnno

	chkiDisabled := label_checkbox.New("Enabled", "Prevent communication with the device")
	inputs[disabled] = &chkiDisabled

	btnCancel := button.New("Cancel")
	inputs[cancel] = &btnCancel
	btnSubmit := button.New("Create")
	inputs[submit] = &btnSubmit

	return &Model{
		ctx:    ctx,
		help:   mir_help.New(keys),
		inputs: inputs,
	}
}

func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		cmds[i] = m.inputs[i].Init()
	}
	cmds = append(cmds, textinput.Blink)
	return tea.Batch(cmds...)

}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))
	switch msg := msg.(type) {
	case button.ButtonPressedMsg:
		if msg.Label == "Cancel" {
			return m, msgs.RouteChangeCmd("/devices")
		} else if msg.Label == "Create" {

		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// TODO submit on cancel or create buttons
		case tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			m.nextInput()
		}

		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()

	}
	m.help, _ = m.help.Update(msg)

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

// TODO error on validation
// TODO components for checkbox and maps
// TODO create and cancel button
// Try with an interface on Focus, Blur and Render
// Needs a component for button, map and checkbox
func (m *Model) View() string {
	var v strings.Builder
	v.WriteString("\n")
	v.WriteString(lipgloss.NewStyle().Bold(true).Render("Create a new Device"))
	v.WriteString("\n")
	v.WriteString(m.inputs[deviceId].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[name].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[description].View())
	v.WriteString("\n\n")
	v.WriteString(m.inputs[disabled].View())
	v.WriteString("\n\n")
	v.WriteString(m.inputs[labels].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[annotations].View())
	v.WriteString("\n\n")
	v.WriteString(m.inputs[cancel].View())
	v.WriteString("  ")
	v.WriteString(m.inputs[submit].View())

	// v.WriteString("\n" + m.deviceIdInput.View() + "\n")
	// if m.deviceIdInput.Err != nil {
	// 	v.WriteString(store.Styles["error"].Render(m.deviceIdInput.Err.Error()))
	// }
	v.WriteString("\n\n\n" + m.help.View())

	return v.String()
}

// nextInput focuses the next input field
func (m *Model) nextInput() {
	m.focused = (m.focused + 1) % len(m.inputs)
}

// prevInput focuses the previous input field
func (m *Model) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = len(m.inputs) - 1
	}
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["search"], k["create"], k["edit"]},
		{k["up"], k["down"]},
	}
}

var keys = keyMap{
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
}

func deviceIdValidator(s string) error {
	return nil
}
