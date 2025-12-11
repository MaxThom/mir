package tui

import (
	"context"

	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/maxthom/mir/internal/ui"
	"github.com/maxthom/mir/internal/ui/tui/components/labelspinner"
	"github.com/maxthom/mir/internal/ui/tui/components/menu"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_configuration "github.com/maxthom/mir/internal/ui/tui/pages/device/cfg"
	device_configuration_response "github.com/maxthom/mir/internal/ui/tui/pages/device/cfg/response"
	device_configuration_values "github.com/maxthom/mir/internal/ui/tui/pages/device/cfg/values"
	device_commands "github.com/maxthom/mir/internal/ui/tui/pages/device/cmd"
	device_command_response "github.com/maxthom/mir/internal/ui/tui/pages/device/cmd/response"
	device_create "github.com/maxthom/mir/internal/ui/tui/pages/device/create"
	device_edit "github.com/maxthom/mir/internal/ui/tui/pages/device/edit"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	device_schema "github.com/maxthom/mir/internal/ui/tui/pages/device/schema"
	device_telemetry "github.com/maxthom/mir/internal/ui/tui/pages/device/tlm"
	"github.com/maxthom/mir/internal/ui/tui/pages/mainmenu"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/internal/ui/tui/styles"
	"github.com/maxthom/mir/pkgs/module/mir"
)

var v strings.Builder

type MirTeaModel interface {
	InitWithData(d any) tea.Cmd
	ResumeWithData(d any) tea.Cmd
	Resume() tea.Cmd
	tea.Model
}

type Model struct {
	ctx          context.Context
	m            *mir.Mir
	cfg          ui.Config
	lblSpinner   labelspinner.Model
	currentRoute menu.OptionValue
	routes       map[string]MirTeaModel
}

func NewModel(ctx context.Context, log zerolog.Logger, m *mir.Mir, cfg ui.Config) *Model {
	log = log.With().Str("page", "router").Logger()
	s := labelspinner.New(" 🛰️ ", styles.Mir.Render("Mir ("+store.MirCtx.Name+")"), spinner.Dot)
	routes := map[string]MirTeaModel{
		"/":                                mainmenu.NewModel(),
		"/devices":                         device_list.NewModel(ctx),
		"/devices/create":                  device_create.NewModel(ctx),
		"/devices/edit":                    device_edit.NewModel(ctx),
		"/devices/schema":                  device_schema.NewModel(ctx),
		"/devices/telemetry":               device_telemetry.NewModel(ctx),
		"/devices/commands":                device_commands.NewModel(ctx),
		"/devices/commands/responses":      device_command_response.NewModel(ctx),
		"/devices/configuration":           device_configuration.NewModel(ctx),
		"/devices/configuration/responses": device_configuration_response.NewModel(ctx),
		"/devices/configuration/values":    device_configuration_values.NewModel(ctx),
		"/twins":                           nil,
		"/telemetry":                       nil,
	}
	return &Model{
		ctx:          ctx,
		currentRoute: "/devices",
		routes:       routes,
		m:            m,
		cfg:          cfg,
		lblSpinner:   s,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Sequence(tea.ClearScreen, tea.Batch(msgs.ReqMsgCmd("connecting to Mir ...standby", msgs.DefaultTimeout), m.connectToMir()))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		store.ScreenWidth = msg.Width
		store.ScreenHeight = msg.Height
	case msgs.ReqMsg:
		var cmd tea.Cmd
		if msg.Timeout != 0 {
			cmd = m.lblSpinner.UpdateLabelWithTimeout(styles.Info.Render(msg.Msg), msg.Timeout)
		} else {
			m.lblSpinner.UpdateLabel(styles.Info.Render(msg.Msg))
		}
		return m, tea.Batch(m.lblSpinner.Start(), cmd)
	case msgs.ResMsg:
		var cmd tea.Cmd
		if msg.Timeout != 0 {
			cmd = m.lblSpinner.UpdateLabelWithTimeout(styles.Info.Render(msg.Msg), msg.Timeout)
		} else {
			m.lblSpinner.UpdateLabel(styles.Info.Render(msg.Msg))
		}
		return m, tea.Batch(m.lblSpinner.Stop(), cmd)
	case msgs.ErrMsg:
		if msg.Timeout != 0 {
			m.lblSpinner.UpdateLabelWithTimeout(styles.Error.Render(msg.Error()), msg.Timeout)
		} else {
			m.lblSpinner.UpdateLabel(styles.Error.Render(msg.Error()))
		}
		return m, tea.Batch(m.lblSpinner.Stop())
	case msgs.RouteChangeMsg:
		m.currentRoute = msg.Route
		if m.routes[m.currentRoute] == nil {
			m.currentRoute = "/"
			return m, m.lblSpinner.UpdateLabelWithTimeout(styles.Error.Render("not implemented yet"), 2*time.Second)
		} else if msg.Data != nil {
			return m, m.routes[m.currentRoute].InitWithData(msg.Data)
		} else {
			return m, m.routes[m.currentRoute].Init()
		}
	case msgs.RouteResumeMsg:
		m.currentRoute = msg.Route
		if m.routes[m.currentRoute] == nil {
			m.currentRoute = "/"
			return m, m.lblSpinner.UpdateLabelWithTimeout(styles.Error.Render("not implemented yet"), 2*time.Second)
		} else if msg.Data == nil {
			return m, m.routes[m.currentRoute].ResumeWithData(msg.Data)
		} else {
			return m, m.routes[m.currentRoute].Resume()
		}
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		case msg.Type == tea.KeyEscape:
			return m, msgs.RouteResume(removeLastSegment(m.currentRoute))
		default:
			rm, cmd := m.routes[m.currentRoute].Update(msg)
			m.routes[m.currentRoute] = rm.(MirTeaModel)
			if cmd != nil {
				return m, cmd
			}
		}
		return m, cmd
	}

	cmds := make([]tea.Cmd, 2)
	var rm tea.Model
	m.lblSpinner, cmds[0] = m.lblSpinner.Update(msg)
	rm, cmds[1] = m.routes[m.currentRoute].Update(msg)
	m.routes[m.currentRoute] = rm.(MirTeaModel)
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	v.Reset()
	v.WriteString(m.lblSpinner.View())
	v.WriteString("\n")
	v.WriteString(m.routes[m.currentRoute].View())
	return v.String()
}

func removeLastSegment(path string) string {
	lastIndex := strings.LastIndex(path, "/")
	if lastIndex == -1 {
		return "/"
	}
	if path[:lastIndex] == "" {
		return "/"
	}
	return path[:lastIndex]
}

func (m *Model) connectToMir() tea.Cmd {
	return func() tea.Msg {
		store.Bus = m.m
		return msgs.ResMsgCmd("connected to Mir", msgs.DefaultTimeout)
	}
}
