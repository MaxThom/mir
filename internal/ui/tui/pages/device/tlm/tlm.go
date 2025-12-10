package device_telemetry

import (
	"context"
	"fmt"
	"sort"
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
	"github.com/maxthom/mir/internal/ui/tui/styles"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l zerolog.Logger
	v strings.Builder
)

const ()

type Model struct {
	ctx       context.Context
	help      mir_help.Model
	devices   []mir_v1.Device
	telemetry []*mir_apiv1.DevicesTelemetry
	menu      group_menu.Model
}

type InputData struct {
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_tlm_list").Logger()

	return &Model{
		ctx:  ctx,
		help: mir_help.New(keys, []string{}, "mir device telemetry"),
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

	return msgs.ListMirDeviceTelemetry(store.Bus, &target)
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
	var tlm tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceTelemetryListedMsg:
		l.Debug().Int("count", len(msg.Telemetry)).Msg("received device telemetry list")
		m.telemetry = msg.Telemetry
		groupchoice := []group_menu.GroupChoice{}
		for _, tlm := range m.telemetry {
			gc := group_menu.GroupChoice{
				Choices: []group_menu.Option{},
			}

			devsTitle := []string{}
			for _, devId := range tlm.Ids {
				if devId.Name == "" && devId.Namespace == "" {
					devsTitle = append(devsTitle, devId.DeviceId)
				} else {
					devsTitle = append(devsTitle, devId.Name+"/"+devId.Namespace)
				}
			}
			if len(devsTitle) > 3 {
				gc.Label = strings.Join(devsTitle[0:3], ", ") + " & " + fmt.Sprintf("%d more", len(devsTitle)-3)
			} else {
				gc.Label = strings.Join(devsTitle, ", ")
			}

			if tlm.Error != "" {
				gc.Choices = append(gc.Choices, group_menu.Option{
					Label:       "error",
					Description: tlm.Error,
				})
			} else if len(tlm.TlmDescriptors) > 0 {
				var sb strings.Builder
				for _, desc := range tlm.TlmDescriptors {
					sb.Reset()
					sb.WriteString("      ")
					sb.WriteString(strings.Join(desc.Fields, "\n      "))
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
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d telemetry fetched", len(msg.Telemetry)), msgs.DefaultTimeout)
	case msgs.EditorFinishedMsg:
		return m, nil
	case tea.KeyMsg:
		m.help, tlm = m.help.Update(msg)
		if msg.String() == "g" {
			i, j := m.menu.GetCursor()
			tlmQuery := m.telemetry[i].TlmDescriptors[j].ExploreQuery
			if tlmQuery != "" {
				if err := grafana.OpenBrowser(grafana.CreateExploreLink(store.MirCtx.Grafana, tlmQuery)); err != nil {
					return m, msgs.ErrCmd(fmt.Errorf("failed to open grafana link: %w", err), 2*time.Second)
				}
			} else {
				return m, msgs.ErrCmd(fmt.Errorf("no grafana query available for this telemetry"), 2*time.Second)
			}
		} else if msg.String() == "q" {
			i, j := m.menu.GetCursor()
			tlmQuery := m.telemetry[i].TlmDescriptors[j].ExploreQuery
			if tlmQuery != "" {
				return m, msgs.OpenEditorCmd(msgs.FileTypeYAML, []byte(tlmQuery), []string{})
			}
		} else {
			m.menu, tlm = m.menu.Update(msg)
		}
		return m, tlm
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")
	header := styles.Help.Bold(false).Render(fmt.Sprintf("Telemetry list for %d devices", len(m.devices)))
	v.WriteString(header + "\n\n")
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
