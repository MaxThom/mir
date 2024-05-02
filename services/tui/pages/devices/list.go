package devices

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	client_core "github.com/maxthom/mir/services/core"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/maxthom/mir/services/tui/store"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields
// TODO search field with / to focus and enter to focus table

var (
	l = log.With().Str("cmp", "root").Logger()
)

type (
	SearchFilterMsg struct {
		filter string
	}
	DeviceFetchedMsg struct {
		devices []*core.Device
	}
)

type Model struct {
	ctx         context.Context
	bus         *bus.BusConn
	help        mir_help.Model
	table       table.Model
	searchInput textinput.Model
}

var styles = map[string]lipgloss.Style{
	"mir":   lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")),
	"table": lipgloss.NewStyle(),
	//BorderStyle(lipgloss.NormalBorder()).
	//BorderForeground(lipgloss.Color("240")),
}

func NewModel(ctx context.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Blur()
	ti.CharLimit = 256
	ti.Width = 50
	ti.ShowSuggestions = true
	ti.SetSuggestions([]string{"search"})

	columns := []table.Column{
		{Title: "", Width: 2}, // Icon with status. online/offline/desabled
		{Title: "id", Width: 10},
		{Title: "name", Width: 20},
		{Title: "labels", Width: 50},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		//BorderStyle(lipgloss.NormalBorder()).
		//BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FF75B7")).
		//Background(lipgloss.Color("57")).
		Bold(false).Italic(true)

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(s),
	)

	return &Model{
		ctx:         ctx,
		help:        mir_help.New(keys),
		table:       t,
		searchInput: ti,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(msgs.ReqMsgCmd("fetching devices..."), fetchMirDevices(store.Bus))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case DeviceFetchedMsg:
		m.table.SetRows(getRows(msg.devices))
		return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.devices)))
	case tea.KeyMsg:
		if m.searchInput.Focused() {
			if msg.Type == tea.KeyEnter {
				m.table.Focus()
				m.searchInput.Blur()
			} else {
				m.searchInput, cmd = m.searchInput.Update(msg)
			}
		} else if m.table.Focused() {
			m.help, cmd = m.help.Update(msg)
			if msg.String() == "q" {
				return m, tea.Quit
			} else if msg.String() == "/" {
				m.table.Blur()
				m.searchInput.Focus()
			} else {
				m.table, cmd = m.table.Update(msg)
			}
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var v strings.Builder
	v.WriteString("\n")
	if m.searchInput.Focused() || m.searchInput.Value() != "" {
		v.WriteString("" + m.searchInput.View())
		v.WriteString("\n")
	}
	v.WriteString(styles["table"].Render(m.table.View() + "\n"))
	v.WriteString("\n\n" + m.help.View())
	return v.String()
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
	"search": key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	"create": key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create"),
	),
	"edit": key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
}

func getRows(devices []*core.Device) []table.Row {
	rows := []table.Row{}
	for _, d := range devices {
		lbls := []string{}
		for k, v := range d.Labels {
			lbls = append(lbls, k+"="+v)
		}
		rows = append(rows, table.Row{
			"🟢", d.DeviceId, d.Name, strings.Join(lbls, ","),
		})
		// ⭕🔴🟢: desabled, offline, online
	}
	return rows
}

func fetchMirDevices(bus *bus.BusConn) tea.Cmd {
	return func() tea.Msg {
		resp, err := client_core.PublishDeviceListRequest(bus, &core.ListDeviceRequest{
			Targets: &core.Targets{},
		})
		if err != nil {
			// TODO move error from cli to next to core client and use it here as well
			return msgs.ErrMsg{Err: err}
		}
		if resp.GetError() != nil {
			return msgs.ErrMsg{Err: fmt.Errorf(resp.GetError().GetMessage())}
		}
		return DeviceFetchedMsg{devices: resp.GetOk().Devices}
	}
}
