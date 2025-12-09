package device_commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/internal/ui/tui/components/group_menu"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	"github.com/maxthom/mir/internal/ui/tui/store"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	l zerolog.Logger
	v strings.Builder
)

const (
	menuOption_device_command_response string = "/devices/commands/responses"
)

type Model struct {
	ctx      context.Context
	help     mir_help.Model
	devices  []mir_v1.Device
	commands []*mir_apiv1.DevicesCommands
	menu     group_menu.Model
	readOnly bool
}

type InputData struct {
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_cmd_list").Logger()

	return &Model{
		ctx:      ctx,
		help:     mir_help.New(keys, []string{}, "mir device commands"),
		menu:     group_menu.New(nil),
		readOnly: true}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	devs, ok := d.([]mir_v1.Device)
	if !ok {
		return tea.Batch(
			msgs.ErrCmd(fmt.Errorf("no device specified"), 2*time.Second),
			msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
		)
	}
	m.devices = devs

	target := mir_apiv1.DeviceTarget{}
	for _, d := range m.devices {
		target.Ids = append(target.Ids, d.Spec.DeviceId)
	}

	return msgs.ListMirDeviceCommands(store.Bus, &target)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		msgs.ErrCmd(fmt.Errorf("no device specified"), 2*time.Second),
		msgs.RouteChangeWithDataCmd("/devices", device_list.InputData{SilentFetch: true}),
	)
}

func (m Model) Resume() tea.Cmd {
	return nil
}

func (m Model) ResumeWithData(d any) tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceCommandListedMsg:
		m.commands = msg.Commands
		groupchoice := []group_menu.GroupChoice{}
		for _, cmd := range m.commands {
			gc := group_menu.GroupChoice{
				Choices: []group_menu.Option{},
			}
			if len(cmd.DevicesNamens) > 3 {
				gc.Label = strings.Join(cmd.DevicesNamens[0:3], ", ") + " & " + strconv.Itoa(len(cmd.DevicesNamens)-3) + " more"
			} else if len(cmd.DevicesNamens) == 0 {
				if len(cmd.DevicesId) > 3 {
					gc.Label = strings.Join(cmd.DevicesId[0:3], ", ") + " & " + strconv.Itoa(len(cmd.DevicesId)-3) + " more"
				} else {
					gc.Label = strings.Join(cmd.DevicesId, ", ")
				}
			} else {
				gc.Label = strings.Join(cmd.DevicesNamens, ", ")
			}

			if cmd.Error != "" {
				gc.Choices = append(gc.Choices, group_menu.Option{
					Label:       "error",
					Description: cmd.Error,
				})
			} else if len(cmd.CmdDescriptors) > 0 {
				var sb strings.Builder
				for _, desc := range cmd.CmdDescriptors {
					sb.Reset()
					if len(desc.Labels) == 0 {
						sb.WriteString("    {}")
					} else {
						sb.WriteString("    {\n      ")
						sb.WriteString(mapToSortedString(desc.Labels, "\n      "))
						sb.WriteString("\n    }")
					}
					gc.Choices = append(gc.Choices, group_menu.Option{
						Label:       desc.Name,
						Description: desc.Error,
						Value:       desc.Name,
						Details:     sb.String(),
					})
				}
			}
			groupchoice = append(groupchoice, gc)
		}
		m.menu = group_menu.New(groupchoice)
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d commands fetched", len(msg.Commands)), msgs.DefaultTimeout)
	case msgs.EditorFinishedMsg:
		if m.readOnly {
			return m, tea.ClearScreen
		}
		i, j := m.menu.GetCursor()
		c := m.commands[i].CmdDescriptors[j]
		t := mir_apiv1.DeviceTarget{
			Ids: m.commands[i].DevicesId,
		}
		devCmds := mir_apiv1.SendCommandRequest{
			Name:            c.Name,
			Payload:         json.RawMessage(msg.Content),
			PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
			Targets:         &t,
		}
		return m, tea.Sequence(tea.ClearScreen, tea.Batch(
			msgs.RouteChangeWithDataCmd(menuOption_device_command_response, &devCmds),
		))
	case tea.KeyMsg:
		m.help, cmd = m.help.Update(msg)
		if msg.String() == "q" || msg.String() == "r" {
			i, j := m.menu.GetCursor()
			cmdDesc := m.commands[i].CmdDescriptors[j]
			query, err := prettyPrintJSON(cmdDesc.Template)
			if err != nil {
				query = err.Error()
			}
			if query != "" {
				headers := []string{}
				if msg.String() == "q" {
					m.readOnly = true
					headers = []string{
						"READ-ONLY MODE: Command will not be sent",
						cmdDesc.Name + "{" + mapToSortedString(cmdDesc.Labels, ", ") + "}",
					}
				} else {
					headers = []string{
						"SEND MODE: Command will be sent",
						cmdDesc.Name + "{" + mapToSortedString(cmdDesc.Labels, ", ") + "}",
					}
					m.readOnly = false
				}
				return m, msgs.OpenEditorCmd(msgs.FileTypeJSON, []byte(query), headers)
			}
		} else if msg.String() == "l" {
			return m, msgs.RouteResume(menuOption_device_command_response)
		} else {
			m.menu, cmd = m.menu.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")

	v.WriteString(m.menu.View())
	v.WriteString(m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["space"], k["send"], k["show"], k["last"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["space"], k["send"], k["show"], k["last"]},
	}
}

var keys = keyMap{
	"space": key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "labels"),
	),
	"show": key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "show template"),
	),
	"send": key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "send command"),
	),
	"last": key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "show last response"),
	),
}

func mapToSortedString(m map[string]string, separator string) string {
	if len(m) == 0 {
		return ""
	}

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted string
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(separator)
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}

func prettyPrintJSON(jsonStr string) (string, error) {
	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}

	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(prettyJSON), nil
}
