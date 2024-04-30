package tui

import (
	"context"

	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/tui/components/labelspinner"
	"github.com/maxthom/mir/services/tui/components/menu"
	"github.com/maxthom/mir/services/tui/msgs"
	"github.com/maxthom/mir/services/tui/pages/devices"
	"github.com/maxthom/mir/services/tui/pages/mainmenu"
	"github.com/nats-io/nats.go"
)

var l zerolog.Logger

type Config struct{}

type TUI struct {
	mirUrl string
}

func NewServer(log zerolog.Logger, mirUrl string) *TUI {
	l = log.With().Str("srv", "tui").Logger()
	return &TUI{mirUrl: mirUrl}
}

func (s *TUI) Launch(ctx context.Context) error {
	p := tea.NewProgram(NewModel(ctx, l, s.mirUrl))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

type connMirCmdMsg struct{}

type Model struct {
	ctx          context.Context
	bus          *bus.BusConn
	mirUrl       string
	lblSpinner   labelspinner.Model
	currentRoute menu.OptionValue
	routes       map[string]tea.Model
}

var styles = map[string]lipgloss.Style{
	"mir":     lipgloss.NewStyle().Foreground(lipgloss.Color("#C26BFF")),
	"error":   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0500")),
	"success": lipgloss.NewStyle().Foreground(lipgloss.Color("#80ff00")),
	"info":    lipgloss.NewStyle().Foreground(lipgloss.Color("#007fff")),
}

func NewModel(ctx context.Context, log zerolog.Logger, mirUrl string) *Model {
	l = log.With().Str("cmp", "tui").Logger()
	s := labelspinner.New(" 🛰️ ", styles["info"].Render("Mir"), spinner.Dot)
	routes := map[string]tea.Model{
		"/":          mainmenu.NewModel(),
		"/devices":   devices.NewModel(ctx, nil),
		"/twins":     nil,
		"/telemetry": nil,
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
	return func() tea.Msg {
		return connMirCmdMsg{}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connMirCmdMsg:
		m.lblSpinner.UpdateLabel("connecting to Mir ...standby")
		return m, tea.Batch(m.lblSpinner.Start(), m.connectToMir(m.mirUrl))
	case msgs.ResMsg:
		cmd := m.lblSpinner.UpdateLabelWithTimeout(styles["success"].Render(msg), 2*time.Second)
		return m, tea.Batch(m.lblSpinner.Stop(), cmd)
	case msgs.ErrMsg:
		m.lblSpinner.UpdateLabel(styles["error"].Render(msg.Error()))
		return m, tea.Batch(m.lblSpinner.Stop())
	case msgs.RouteChangeMsg:
		m.currentRoute = msg.Route
		if m.routes[m.currentRoute] == nil {
			m.currentRoute = "/"
			return m, m.lblSpinner.UpdateLabelWithTimeout(styles["error"].Render("not implemented yet"), 2*time.Second)
		}
		return m, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		case msg.Type == tea.KeyCtrlC || msg.String() == "q":
			return m, tea.Quit
		case msg.Type == tea.KeyEscape:
			return m, msgs.RouteChangeCmd(removeLastSegment(m.currentRoute))
		default:
			m.routes[m.currentRoute], cmd = m.routes[m.currentRoute].Update(msg)
			if cmd != nil {
				return m, cmd
			}
		}
		return m, cmd
	default:
		var cmd tea.Cmd
		m.lblSpinner, cmd = m.lblSpinner.Update(msg)
		if cmd != nil {
			return m, cmd
		}
		m.routes[m.currentRoute], cmd = m.routes[m.currentRoute].Update(msg)
		return m, cmd
	}
}

func (m *Model) View() string {
	var v strings.Builder
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

		return msgs.ResMsg("connected to Mir")
	}
}
