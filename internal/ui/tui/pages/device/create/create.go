package device_create

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/ui/tui/components/form"
	"github.com/maxthom/mir/internal/ui/tui/components/form/button"
	"github.com/maxthom/mir/internal/ui/tui/components/form/label_checkbox"
	"github.com/maxthom/mir/internal/ui/tui/components/form/label_textbox"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/internal/ui/tui/styles"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// BUG blink on text input
// BUG array out of bound on current suggestion
// BUG set cursor on position 0 for checkbox

var (
	l zerolog.Logger
)

const (
	deviceId = iota
	name
	namespace
	description
	disabled
	labels
	annotations
	cancel
	submit
	submitRandom
)

var (
	v strings.Builder
)

type Model struct {
	ctx          context.Context
	help         mir_help.Model
	inputs       []form.Control
	focused      int
	displayError bool
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_create").Logger()
	inputs := make([]form.Control, 10)
	tiId := label_textbox.New("Unique ID  ", "Suggestions are existing IDs", form.MirValidators(form.WithMandatoryValidator()))
	tiId.CharLimit = 50
	tiId.Width = 60
	tiId.ShowSuggestions = true
	tiId.Focus()
	inputs[deviceId] = &tiId

	tiNm := label_textbox.New("Name       ", "", nil)
	tiNm.CharLimit = 50
	tiNm.Width = 60
	inputs[name] = &tiNm

	tiNs := label_textbox.New("Namespace  ", "", nil)
	tiNs.CharLimit = 50
	tiNs.Width = 60
	inputs[namespace] = &tiNs

	tiDesc := label_textbox.New("Description", "", nil)
	tiDesc.CharLimit = 50
	tiDesc.Width = 60
	inputs[description] = &tiDesc

	tiLbls := label_textbox.New("Labels     ", "Set of indexed key value pairs to identify the device <k1=v1;k2=v2>", form.MirValidators(form.WithKeyValueMapValidator()))
	tiLbls.CharLimit = 50
	tiLbls.Width = 60
	tiLbls.ShowSuggestions = true
	inputs[labels] = &tiLbls

	tiAnno := label_textbox.New("Annotations", "Set of key value pairs to add information on the device <k1=v1;k2=v2>", form.MirValidators(form.WithKeyValueMapValidator()))
	tiAnno.CharLimit = 50
	tiAnno.Width = 60
	tiAnno.ShowSuggestions = true
	inputs[annotations] = &tiAnno

	chkiDisabled := label_checkbox.New("Enabled    ", "Prevent communication with the device", true, nil)
	inputs[disabled] = &chkiDisabled

	btnCancel := button.New("Previous", "", nil)
	inputs[cancel] = &btnCancel
	btnSubmit := button.New("Create", "", nil)
	inputs[submit] = &btnSubmit
	btnSubmitWithRandom := button.New("Create with random uuid", "", nil)
	inputs[submitRandom] = &btnSubmitWithRandom

	tooltips := []string{}
	for _, v := range inputs {
		t := v.GetTooltip()
		if t != "" {
			tooltips = append(tooltips, v.GetLabel()+" > "+t)
		}
	}

	return &Model{
		ctx:    ctx,
		help:   mir_help.New(keys, tooltips, "mir device create"),
		inputs: inputs,
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	return m.Init()
}

func (m *Model) Init() tea.Cmd {
	m.inputs[deviceId].(*label_textbox.Model).SetSuggestions(store.GetDeviceIdSuggestions(store.Devices))
	m.inputs[namespace].(*label_textbox.Model).SetSuggestions(store.GetNamespaceSuggestions(store.Devices))
	m.inputs[labels].(*label_textbox.Model).SetSuggestions(store.GetLabelsSuggestions(store.Devices))
	m.inputs[annotations].(*label_textbox.Model).SetSuggestions(store.GetAnnotationsSuggestions(store.Devices))
	m.displayError = false
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
		if msg.Label == "Previous" {
			return m, msgs.RouteChangeCmd("/devices")
		} else if msg.Label == "Create" {
			// We want the error to be displayed
			// only after first attempt of Create
			// else, they are annoying
			m.displayError = true
			formInError := false
			for _, i := range m.inputs {
				if i.GetErr() != nil {
					formInError = true
				}
			}
			if !formInError {
				req := &core_api.CreateDeviceRequest{
					DeviceId:    m.inputs[deviceId].GetValue(),
					Name:        m.inputs[name].GetValue(),
					Namespace:   m.inputs[namespace].GetValue(),
					Disabled:    !boolStringToBool(m.inputs[disabled].GetValue()),
					Labels:      keyValueStringToMap(m.inputs[labels].GetValue()),
					Annotations: keyValueStringToMap(m.inputs[annotations].GetValue()),
				}
				req.Annotations["mir/device/description"] = m.inputs[description].GetValue()
				return m, tea.Batch(msgs.ReqMsgCmd("creating device..."), msgs.CreateMirDevice(store.Bus, req))
			}
		} else if msg.Label == "Create with random uuid" {
			t, err := uuid.NewRandom()
			if err != nil {
				return m, msgs.ErrCmd(err, 2*time.Second)
			}
			m.inputs[deviceId].(*label_textbox.Model).SetValue(t.String())

			return m, m.inputs[submit].(*button.Model).ButtonPressCmd()
		}
	case msgs.DeviceCreatedMsg:
		s := ""
		if len(msg.Devices) == 1 {
			s = fmt.Sprintf("device '%s' created", msg.Devices[0].Spec.DeviceId)
		} else {
			s = fmt.Sprintf("%d devices created", len(msg.Devices))
		}
		return m, tea.Batch(msgs.ResMsgCmd(s))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN:
			// TODO BUG FIX in bubble repo, make a pr
			// The method 'current suggestion' crashes if no suggestion
			// in the meantime, we can complete suggestion.
			//if l, ok := m.inputs[m.focused].(*label_textbox.Model); ok {
			//	if l.CurrentSuggestion() == "" {
			//		m.nextInput()
			//	}
			//} else {
			m.nextInput()
			//}
		}

		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.inputs[m.focused].Focus()

	}
	m.help, _ = m.help.Update(msg)
	m.updateCli()

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")
	v.WriteString(lipgloss.NewStyle().Bold(true).Render("Create a new Device"))
	v.WriteString("\n")
	v.WriteString(m.inputs[deviceId].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[name].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[namespace].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[description].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[disabled].View())
	v.WriteString("\n\n")
	v.WriteString(m.inputs[labels].View())
	v.WriteString("\n")
	v.WriteString(m.inputs[annotations].View())
	v.WriteString("\n\n")
	v.WriteString(m.inputs[cancel].View())
	v.WriteString("  ")
	v.WriteString(m.inputs[submit].View())
	v.WriteString("  ")
	v.WriteString(m.inputs[submitRandom].View())
	v.WriteString("\n\n")

	if m.displayError {
		addLines := false
		for _, i := range m.inputs {
			if i.GetErr() != nil {
				addLines = true
				v.WriteString(styles.Error.Render(i.GetLabel()))
				v.WriteString(styles.Error.Render(" > "))
				v.WriteString(styles.Error.Render(i.GetErr().Error()))
				v.WriteString("\n")
			}
		}
		if addLines {
			v.WriteString("\n")
		}
	}

	v.WriteString(m.help.View())

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

func (m *Model) updateCli() {
	cli := []string{"mir device create"}
	if m.inputs[deviceId].GetValue() != "" {
		cli = append(cli, "--id \""+m.inputs[deviceId].GetValue()+"\"")
	}

	if m.inputs[name].GetValue() != "" {
		cli = append(cli, "--name \""+m.inputs[name].GetValue()+"\"")
	}

	if m.inputs[namespace].GetValue() != "" {
		cli = append(cli, "--namespace \""+m.inputs[namespace].GetValue()+"\"")
	}

	if m.inputs[description].GetValue() != "" {
		cli = append(cli, "--desc \""+m.inputs[description].GetValue()+"\"")
	}

	if m.inputs[labels].GetValue() != "" {
		cli = append(cli, "--labels \""+m.inputs[labels].GetValue()+"\"")
	}

	if m.inputs[annotations].GetValue() != "" {
		cli = append(cli, "--anno \""+m.inputs[annotations].GetValue()+"\"")
	}

	if m.inputs[disabled].GetValue() == "false" {
		cli = append(cli, "--disabled")
	}

	m.help.UpdateCli(strings.Join(cli, " "))
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

func boolStringToBool(s string) bool {
	b, e := strconv.ParseBool(s)
	if e != nil {
		return false
	}
	return b
}

func keyValueStringToMap(s string) map[string]string {
	m := map[string]string{}
	if s == "" {
		return m
	}

	pairs := strings.Split(s, ";")
	for _, p := range pairs {
		kv := strings.Split(p, "=")
		if len(kv) >= 2 {
			m[kv[0]] = kv[1]
		} else if len(kv) == 1 {
			m[kv[0]] = ""
		}
	}

	return m
}
