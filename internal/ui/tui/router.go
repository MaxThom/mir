package tui

import (
	"context"

	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"

	"github.com/charmbracelet/bubbles/spinner"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/ui/tui/components/labelspinner"
	"github.com/maxthom/mir/internal/ui/tui/components/menu"
	"github.com/maxthom/mir/internal/ui/tui/msgs"
	device_create "github.com/maxthom/mir/internal/ui/tui/pages/device/create"
	device_edit "github.com/maxthom/mir/internal/ui/tui/pages/device/edit"
	device_list "github.com/maxthom/mir/internal/ui/tui/pages/device/list"
	"github.com/maxthom/mir/internal/ui/tui/pages/mainmenu"
	"github.com/maxthom/mir/internal/ui/tui/store"
	"github.com/maxthom/mir/internal/ui/tui/styles"
	"github.com/nats-io/nats.go"
)

var v strings.Builder

type MirTeaModel interface {
	InitWithData(d any) tea.Cmd
	tea.Model
}

type Model struct {
	ctx          context.Context
	bus          *bus.BusConn
	mirUrl       string
	lblSpinner   labelspinner.Model
	currentRoute menu.OptionValue
	routes       map[string]MirTeaModel
}

func NewModel(ctx context.Context, log zerolog.Logger, mirUrl string) *Model {
	l = log.With().Str("page", "router").Logger()
	s := labelspinner.New(" 🛰️ ", styles.Mir.Render("Mir"), spinner.Dot)
	routes := map[string]MirTeaModel{
		"/":               mainmenu.NewModel(),
		"/devices":        device_list.NewModel(ctx),
		"/devices/create": device_create.NewModel(ctx),
		"/devices/edit":   device_edit.NewModel(ctx),
		"/twins":          nil,
		"/telemetry":      nil,
	}
	return &Model{
		ctx:          ctx,
		currentRoute: "/",
		routes:       routes,
		mirUrl:       mirUrl,
		lblSpinner:   s,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(msgs.ReqMsgCmd("connecting to Mir ...standby"), m.connectToMir(m.mirUrl))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		store.ScreenWidth = msg.Width
		store.ScreenHeight = msg.Height
	case msgs.ReqMsg:
		m.lblSpinner.UpdateLabel(styles.Info.Render(string(msg)))
		return m, m.lblSpinner.Start()
	case msgs.ResMsg:
		cmd := m.lblSpinner.UpdateLabelWithTimeout(styles.Info.Render(msg), 2*time.Second)
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
		} else {
			// TODO add route param to say if init or not
			return m, m.routes[m.currentRoute].InitWithData(msg.Data)
		}
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		case msg.Type == tea.KeyEscape:
			return m, msgs.RouteChangeCmd(removeLastSegment(m.currentRoute))
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

func (m *Model) connectToMir(url string) tea.Cmd {
	return func() tea.Msg {
		b, err := bus.New(url,
			bus.WithReconnHandler(func(nc *nats.Conn) {
				l.Warn().Msg("reconnected to " + nc.ConnectedUrl())
			}),
			bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
				l.Warn().Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
			}),
			bus.WithClosedHandler(func(nc *nats.Conn) {
				l.Warn().Msg("connection to %v closed " + nc.ConnectedUrl())
			}))
		if err != nil {
			return msgs.ErrMsg{Err: err}
		}
		m.bus = b
		store.Bus = b

		return msgs.ResMsg("connected to Mir")
	}
}
