package device_list

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	mir_help "github.com/maxthom/mir/services/tui/components/help"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/maxthom/mir/services/tui/store"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l                               = log.With().Str("page", "device_list").Logger()
	menuOption_device_create string = "/devices/create"
	menuOption_device_edit   string = "/devices/edit"
	v                        strings.Builder
)

const (
	tableColStatus   = 0
	tableColDeviceID = 1
	tableColName     = 2
	tableColLabels   = 3
)

type Model struct {
	ctx         context.Context
	help        mir_help.Model
	table       table.Model
	searchInput textinput.Model
	deleteInput textinput.Model
	tableRowAll []table.Row
}

func NewModel(ctx context.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = "Search"
	ti.Blur()
	ti.CharLimit = 256
	ti.Width = 50
	ti.ShowSuggestions = true

	delti := textinput.New()
	delti.Placeholder = "yes|no"
	delti.Blur()
	delti.CharLimit = 50
	delti.Width = 50
	delti.ShowSuggestions = true
	delti.SetSuggestions([]string{"yes", "no"})
	delti.Prompt = "Confirm device deletion? > "

	columns := []table.Column{
		{Title: "", Width: 2}, // Icon with status. online/offline/desabled
		{Title: "id", Width: 10},
		{Title: "name", Width: 20},
		{Title: "labels", Width: 50},
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FF75B7")).
		Bold(false).Italic(true)

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
		table.WithStyles(s),
	)

	return &Model{
		ctx:         ctx,
		help:        mir_help.New(keys, []string{}),
		table:       t,
		searchInput: ti,
		deleteInput: delti,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(msgs.ReqMsgCmd("fetching devices..."), msgs.ListMirDevices(store.Bus))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case msgs.DeviceListedMsg:
		store.Devices = msg.Devices
		var suggestions []string
		m.tableRowAll, suggestions = devicesToRows(msg.Devices)
		m.searchInput.SetSuggestions(suggestions)
		m.table.SetRows(m.tableRowAll)
		if !msg.NoToast {
			return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.Devices)))
		}
	case msgs.DeviceDeleteMsg:
		rsp := "device deleted"
		if len(msg.Devices) > 0 {
			rsp = fmt.Sprintf("device '%s' deleted", msg.Devices[0].DeviceId)
		}
		return m, tea.Batch(msgs.ListMirDevicesSilently(store.Bus), msgs.ResMsgCmd(rsp))
	case tea.KeyMsg:
		if m.searchInput.Focused() {
			if msg.Type == tea.KeyEnter {
				m.table.Focus()
				m.searchInput.Blur()
			} else {
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.table.SetRows(filterTableRows(m.tableRowAll, m.searchInput.Value()))
			}
		} else if m.deleteInput.Focused() {
			if msg.Type == tea.KeyEnter {
				m.table.Focus()
				m.deleteInput.Blur()
				if m.deleteInput.Value() != "yes" && m.deleteInput.Value() != "y" {
					return m, nil
				}

				device, ok := rowToDevice(m.table.SelectedRow())
				if !ok {
					return m, msgs.ErrCmd(fmt.Errorf("no device selected"), 2*time.Second)
				}
				return m, tea.Batch(
					msgs.ReqMsgCmd("deleting device "+device.DeviceId+"..."),
					msgs.DeleteMirDevice(store.Bus, &core.DeleteDeviceRequest{
						Targets: &core.Targets{
							Ids: []string{device.DeviceId},
						},
					}))
			} else {
				m.deleteInput, cmd = m.deleteInput.Update(msg)
			}
		} else if m.table.Focused() {
			m.help, cmd = m.help.Update(msg)
			if msg.String() == "q" {
				return m, tea.Quit
			} else if msg.String() == "/" {
				m.table.Blur()
				m.searchInput.Focus()
			} else if msg.String() == "c" {
				return m, msgs.RouteChangeCmd(menuOption_device_create)
			} else if msg.String() == "e" {
				return m, msgs.RouteChangeCmd(menuOption_device_edit)
			} else if msg.String() == "x" {
				m.table.Blur()
				m.deleteInput.SetValue("")
				m.deleteInput.Focus()
			} else {
				m.table, cmd = m.table.Update(msg)
			}
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString("\n")
	if m.searchInput.Focused() || m.searchInput.Value() != "" {
		v.WriteString("" + m.searchInput.View())
		v.WriteString("\n")
	} else if m.deleteInput.Focused() {
		v.WriteString("" + m.deleteInput.View())
		v.WriteString("\n")
	}

	v.WriteString(m.table.View() + "\n")
	v.WriteString("\n\n" + m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["search"], k["create"], k["edit"], k["delete"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["search"], k["create"]},
		{k["edit"], k["delete"]},
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
	"delete": key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete"),
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

func devicesToRows(devices []*core.Device) ([]table.Row, []string) {
	rows := []table.Row{}
	suggestions := []string{}
	for _, d := range devices {
		lbls := []string{}
		lblKeys := []string{}
		for k := range d.Labels {
			lblKeys = append(lblKeys, k)
		}
		sort.Strings(lblKeys)
		for _, k := range lblKeys {
			lbls = append(lbls, k+"="+d.Labels[k])
		}

		status := "🟢"
		if d.Disabled {
			status = "⭕"
		} else if !d.Online {
			status = "🔴"
		}
		rows = append(rows, table.Row{
			status, d.DeviceId, d.Name, strings.Join(lbls, ","),
		})
		suggestions = append(suggestions, d.Name, d.DeviceId)
		suggestions = append(suggestions, lbls...)
	}
	// Sort the rows by labels then name if equal
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i][3] > rows[j][3] {
			return false
		} else if rows[i][3] < rows[j][3] {
			return true
		} else {
			if rows[i][2] > rows[j][2] {
				return false
			} else if rows[i][2] < rows[j][2] {
				return true
			}
		}
		return false
	})
	return rows, suggestions
}

func filterTableRows(rows []table.Row, filter string) []table.Row {
	filteredRows := []table.Row{}
	for _, r := range rows {
		for _, c := range r {
			if strings.Contains(c, filter) {
				filteredRows = append(filteredRows, r)
				break
			}
		}
	}
	return filteredRows
}

func rowToDevice(r table.Row) (*core.Device, bool) {
	if len(r) < 2 {
		return nil, false
	}
	id := r[tableColDeviceID]
	if store.Devices != nil {
		for _, i := range store.Devices {
			if id == i.DeviceId {
				return i, true
			}
		}
	}
	return nil, false
}
