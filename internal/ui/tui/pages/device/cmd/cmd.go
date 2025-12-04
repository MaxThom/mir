package device_commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/maxthom/mir/internal/libs/external/grafana"
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

const ()

type Model struct {
	ctx      context.Context
	help     mir_help.Model
	devices  []mir_v1.Device
	commands []*mir_apiv1.DevicesCommands
	menu     group_menu.Model
}

type InputData struct {
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_tlm_list").Logger()

	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, "mir device commands"),
		menu: group_menu.New(nil),
	}
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceCommandListedMsg:
		l.Debug().Int("count", len(msg.Commands)).Msg("received device commands list")
		m.commands = msg.Commands
		groupchoice := []group_menu.GroupChoice{}
		l.Info().Any("test", m.commands).Msg("asd")
		for _, cmd := range m.commands {
			gc := group_menu.GroupChoice{
				Choices: []group_menu.Option{},
			}
			if len(cmd.DevicesNamens) > 3 {
				gc.Label = strings.Join(cmd.DevicesNamens[0:3], ", ") + " & " + strconv.Itoa(len(cmd.DevicesNamens)-3) + " more"
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
					sb.WriteString("      ")
					// sb.WriteString(strings.Join(desc.Fields, "\n      "))
					gc.Choices = append(gc.Choices, group_menu.Option{
						Label:       fmt.Sprintf("%s{%s}", desc.Name, mapToSortedString(desc.Labels)),
						Description: desc.Error,
						Value:       desc.Name,
						Details:     sb.String(),
					})
				}
			}
			groupchoice = append(groupchoice, gc)
		}
		m.menu = group_menu.New(groupchoice)
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d commands fetched", len(msg.Commands)))
	case msgs.EditorFinishedMsg:
		return m, nil
	case tea.KeyMsg:
		m.help, cmd = m.help.Update(msg)
		if msg.String() == "g" {
			// i, j := m.menu.GetCursor()
			tlmQuery := "" // m.commands[i].TlmDescriptors[j].ExploreQuery
			if tlmQuery != "" {
				if err := grafana.OpenBrowser(grafana.CreateExploreLink(store.MirCtx.Grafana, tlmQuery)); err != nil {
					return m, msgs.ErrCmd(fmt.Errorf("failed to open grafana link: %w", err), 2*time.Second)
				}
			} else {
				return m, msgs.ErrCmd(fmt.Errorf("no grafana query available for this telemetry"), 2*time.Second)
			}
		} else if msg.String() == "q" {
			// i, j := m.menu.GetCursor()
			tlmQuery := "" // km.telemetry[i].TlmDescriptors[j].ExploreQuery
			if tlmQuery != "" {
				return m, msgs.OpenEditorCmd(msgs.FileTypeYAML, []byte(tlmQuery), []string{})
			}
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

	// q := ""
	// i, j := m.menu.GetCursor()
	// if len(m.telemetry) > i && len(m.telemetry[i].TlmDescriptors) > j {
	// TODO in a future, the side panel will be a tlm chart
	// q = m.telemetry[i].TlmDescriptors[j].ExploreQuery
	// q = "{" + mapToSortedString(m.telemetry[i].TlmDescriptors[j].Labels) + "}\n"
	// q += lipgloss.NewStyle().Bold(true).Render("Fields")
	// q += "\n• " + strings.Join(m.telemetry[i].TlmDescriptors[j].Fields, "\n• ")
	// q = styles.SidePanel.Render(q)
	// }
	// v.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, m.menu.View(), "          ", q))

	v.WriteString(m.menu.View())
	v.WriteString(m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["space"], k["grafana"], k["query"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["space"], k["grafana"], k["query"]},
	}
}

var keys = keyMap{
	"space": key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "fields"),
	),
	"grafana": key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "grafana"),
	),
	"query": key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "query"),
	),
}

func mapToSortedString(m map[string]string) string {
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
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}
