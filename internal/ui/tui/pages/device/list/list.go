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
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mir_help "github.com/maxthom/mir/internal/ui/tui/components/help"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IDEA wide option that show more fields

var (
	l                           zerolog.Logger
	menuOption_device_create    string = "/devices/create"
	menuOption_device_edit      string = "/devices/edit"
	menuOption_device_schema    string = "/devices/schema"
	menuOption_device_telemetry string = "/devices/telemetry"
	v                           strings.Builder
)

const (
	tableColStatus    = 0
	tableColChecked   = 1
	tableColDeviceID  = 2
	tableColName      = 3
	tableColNamespace = 4
	tableColLabels    = 5

	refreshInterval = time.Second * 10
)

type Model struct {
	ctx         context.Context
	help        mir_help.Model
	table       table.Model
	searchInput textinput.Model
	deleteInput textinput.Model
	tableRowAll []table.Row
	checkedRows map[string]bool
	timer       timer.Model
}

type InputData struct {
	SilentFetch bool
}

func NewModel(ctx context.Context) *Model {
	l = log.With().Str("page", "device_list").Logger()
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
		{Title: "", Width: 2}, // Icon with checked.
		{Title: "id", Width: 20},
		{Title: "name", Width: 25},
		{Title: "namespace", Width: 25},
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
		table.WithHeight(10),
		table.WithStyles(s),
	)

	return &Model{
		ctx:         ctx,
		help:        mir_help.New(keys, []string{}, "mir device list"),
		table:       t,
		searchInput: ti,
		deleteInput: delti,
		timer:       timer.New(refreshInterval),
		checkedRows: make(map[string]bool),
	}
}

func (m *Model) InitWithData(d any) tea.Cmd {
	m.table.Focus()
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	if a, ok := d.(InputData); ok {
		if a.SilentFetch {
			return tea.Batch(m.timer.Init(), msgs.ListMirDevicesSilently(store.Bus))
		} else {
			return m.Init()
		}
	} else if d != nil {
		e := fmt.Errorf("can't assert data on route init")
		l.Error().Err(e).Msg("")
		return tea.Batch(m.timer.Init(), msgs.ErrCmd(e, 2*time.Second))
	}
	return m.Init()
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.timer.Init(), msgs.ReqMsgCmd("fetching devices..."), msgs.ListMirDevices(store.Bus))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if store.ScreenHeight-10 < len(m.table.Rows()) {
			m.table.SetHeight(store.ScreenHeight - 10)
		}
		return m, nil
	case msgs.DeviceListedMsg:
		store.Devices = msg.Devices
		var suggestions []string
		m.tableRowAll, suggestions = m.devicesToRows(msg.Devices)
		m.searchInput.SetSuggestions(suggestions)
		rows := filterTableRows(m.tableRowAll, m.searchInput.Value())
		if store.ScreenHeight-10 < len(rows) {
			m.table.SetHeight(store.ScreenHeight - 10)
		} else {
			m.table.SetHeight(len(rows))
		}
		m.table.SetRows(rows)
		if !msg.NoToast {
			return m, msgs.ResMsgCmd(fmt.Sprintf("%d devices fetched", len(msg.Devices)))
		}
	case msgs.DeviceDeleteMsg:
		rsp := "device deleted successfully"
		if len(msg.Devices) > 0 {
			rsp = fmt.Sprintf("device '%s' deleted", msg.Devices[0].Spec.DeviceId)
		}
		return m, tea.Batch(msgs.ListMirDevicesSilently(store.Bus), msgs.ResMsgCmd(rsp))
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd
	case timer.TimeoutMsg:
		m.timer = timer.New(refreshInterval)
		return m, tea.Batch(m.timer.Init(), msgs.ListMirDevicesSilently(store.Bus))
	case tea.KeyMsg:
		if m.searchInput.Focused() {
			if msg.Type == tea.KeyEnter {
				m.table.Focus()
				m.searchInput.Blur()

				m.checkedRows = make(map[string]bool)
				rows := m.table.Rows()
				if m.searchInput.Value() != "" {
					for i, r := range rows {
						m.checkedRows[r[tableColDeviceID]] = true
						rows[i][tableColChecked] = "✔"
					}
				} else {
					for i, r := range rows {
						delete(m.checkedRows, r[tableColDeviceID])
						rows[i][tableColChecked] = ""
					}
				}
				m.table.SetRows(rows)
			} else {
				m.searchInput, cmd = m.searchInput.Update(msg)
				rows := filterTableRows(m.tableRowAll, m.searchInput.Value())
				if store.ScreenHeight-10 < len(rows) {
					m.table.SetHeight(store.ScreenHeight - 10)
				} else {
					m.table.SetHeight(len(rows))
				}
				m.table.SetRows(rows)
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
					msgs.ReqMsgCmd("deleting device "+device.Spec.DeviceId+"..."),
					msgs.DeleteMirDevice(store.Bus, mir_v1.DeviceTarget{
						Ids: []string{device.Spec.DeviceId},
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
				device, ok := rowToDevice(m.table.SelectedRow())
				if !ok {
					return m, msgs.ErrCmd(fmt.Errorf("no device selected"), 2*time.Second)
				}
				return m, msgs.RouteChangeWithDataCmd(menuOption_device_edit, device)
			} else if msg.String() == "s" {
				device, ok := rowToDevice(m.table.SelectedRow())
				if !ok {
					return m, msgs.ErrCmd(fmt.Errorf("no device selected"), 2*time.Second)
				}
				return m, msgs.RouteChangeWithDataCmd(menuOption_device_schema, device)
			} else if msg.String() == "x" {
				m.table.Blur()
				m.deleteInput.SetValue("")
				m.deleteInput.Focus()
			} else if msg.String() == tea.KeySpace.String() {
				rows := m.table.Rows()
				if checked, ok := m.checkedRows[m.table.SelectedRow()[tableColDeviceID]]; ok && checked {
					m.checkedRows[m.table.SelectedRow()[tableColDeviceID]] = false
					delete(m.checkedRows, m.table.SelectedRow()[tableColDeviceID])
					rows[m.table.Cursor()][tableColChecked] = ""
				} else {
					m.checkedRows[m.table.SelectedRow()[tableColDeviceID]] = true
					rows[m.table.Cursor()][tableColChecked] = "✔"
				}
				m.table.SetRows(rows)
			} else if msg.String() == "t" {
				devices, ok := m.getSelectedDevices()
				if !ok {
					return m, msgs.ErrCmd(fmt.Errorf("no device selected"), 2*time.Second)
				}
				return m, msgs.RouteChangeWithDataCmd(menuOption_device_telemetry, devices)

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

	v.WriteString(m.table.View())
	v.WriteString("\n")
	v.WriteString(m.help.View())
	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["search"], k["create"], k["edit"], k["delete"], k["schema"], k["tlm"], k["cmd"], k["cfg"]}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["search"], k["space"]},
		{k["create"], k["edit"], k["delete"]},
		{k["schema"], k["tlm"], k["cmd"], k["cfg"]},
		{k["up"], k["down"]},
	}
}

var keys = keyMap{
	"space": key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select/deselect"),
	),
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
	"schema": key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "schema"),
	),
	"up": key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	"down": key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	"tlm": key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "telemetry"),
	),
	"cmd": key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "command"),
	),
	"cfg": key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "config"),
	),
}

func (m *Model) devicesToRows(devices []mir_v1.Device) ([]table.Row, []string) {
	rows := []table.Row{}
	suggestions := []string{}
	for _, d := range devices {
		lbls := []string{}
		lblKeys := []string{}
		for k := range d.Meta.Labels {
			lblKeys = append(lblKeys, k)
		}
		sort.Strings(lblKeys)
		for _, k := range lblKeys {
			lbls = append(lbls, k+"="+d.Meta.Labels[k])
		}

		status := "🟢"
		if *d.Spec.Disabled {
			status = "⭕"
		} else if !*d.Status.Online {
			status = "🔴"
		}

		checked := ""
		if _, ok := m.checkedRows[d.Spec.DeviceId]; ok {
			checked = "✔"
		}

		rows = append(rows, table.Row{
			status, checked, d.Spec.DeviceId, d.Meta.Name, d.Meta.Namespace, strings.Join(lbls, ","),
		})
		suggestions = append(suggestions, d.Spec.DeviceId, d.Meta.Name, d.Meta.Namespace)
		suggestions = append(suggestions, lbls...)
	}
	// Sort the rows by namespace, then labels then name
	// Empty value are at the bottom
	sort.SliceStable(rows, func(i, j int) bool {
		// Namespace
		if rows[i][tableColNamespace] > rows[j][tableColNamespace] {
			if rows[j][tableColNamespace] == "" {
				return true
			}
			return false
		} else if rows[i][tableColNamespace] < rows[j][tableColNamespace] {
			if rows[i][tableColNamespace] == "" {
				return false
			}
			return true
		}
		// Labels
		if rows[i][tableColLabels] > rows[j][tableColLabels] {
			if rows[j][tableColLabels] == "" {
				return true
			}
			return false
		} else if rows[i][tableColLabels] < rows[j][tableColLabels] {
			if rows[i][tableColLabels] == "" {
				return false
			}
			return true
		}
		// Name
		if rows[i][tableColName] > rows[j][tableColName] {
			if rows[j][tableColName] == "" {
				return true
			}
			return false
		} else if rows[i][tableColName] < rows[j][tableColName] {
			if rows[i][tableColName] == "" {
				return false
			}
			return true
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

func (m *Model) getSelectedDevices() ([]mir_v1.Device, bool) {
	if len(m.checkedRows) == 0 {
		d, ok := rowToDevice(m.table.SelectedRow())
		return []mir_v1.Device{d}, ok
	} else {
		deviceIds := make([]string, 0, len(m.checkedRows))
		for deviceId := range m.checkedRows {
			deviceIds = append(deviceIds, deviceId)
		}
		return deviceIdsToDevices(deviceIds)
	}
}

func rowToDevice(r table.Row) (mir_v1.Device, bool) {
	if len(r) < 2 {
		return mir_v1.Device{}, false
	}
	id := r[tableColDeviceID]
	if store.Devices != nil {
		for _, i := range store.Devices {
			if id == i.Spec.DeviceId {
				return i, true
			}
		}
	}
	return mir_v1.Device{}, false
}

func deviceIdsToDevices(ids []string) ([]mir_v1.Device, bool) {
	devs := []mir_v1.Device{}
	if store.Devices != nil {
		for _, id := range ids {
			for _, d := range store.Devices {
				if id == d.Spec.DeviceId {
					devs = append(devs, d)
				}
			}
		}
	}
	return devs, len(devs) > 0
}
