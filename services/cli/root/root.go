package root

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bus "github.com/maxthom/mir/libs/external/natsio"
	mir_help "github.com/maxthom/mir/services/cli/components/help"
	"github.com/maxthom/mir/services/cli/components/labelspinner"
	"github.com/maxthom/mir/services/cli/components/menu"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

var (
	menuOption_devices   menu.OptionValue = "devices"
	menuOption_twins     menu.OptionValue = "twins"
	menuOption_telemetry menu.OptionValue = "telemetry"
)

type Model struct {
	ctx          context.Context
	bus          *bus.BusConn
	mirUrl       string
	topLbl       string
	err          error
	lblSpinner   labelspinner.Model
	mainMenu     menu.Model
	currentRoute menu.OptionValue
	help         mir_help.Model
}

var styles = map[string]lipgloss.Style{
	"mir": lipgloss.NewStyle().Foreground(lipgloss.Color("#C26BFF")),
}

func NewModel(ctx context.Context, log zerolog.Logger, mirUrl string) *Model {
	l = log.With().Str("cmp", "root").Logger()
	s := labelspinner.New("", spinner.Dot)
	mm := menu.New([]menu.Option{
		{
			Value:       menuOption_devices,
			Label:       "devices",
			Description: "create and manage a fleet of Mir devices",
		},
		{
			Value:       menuOption_twins,
			Label:       "(wip) twins",
			Description: "manage a set of default twins template for devices configuration",
		},
		{
			Value:       menuOption_telemetry,
			Label:       "(wip) telemetry",
			Description: "upload new telemetry schema",
		},
	})
	return &Model{
		ctx:        ctx,
		mirUrl:     mirUrl,
		lblSpinner: s,
		mainMenu:   mm,
		help:       mir_help.New(keys),
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
	case resMsg:
		m.lblSpinner.UpdateLabel(msg)
		return m, m.lblSpinner.Stop()
	case errMsg:
		m.lblSpinner.UpdateLabel(msg.Error())
		return m, m.lblSpinner.Stop()
	case menu.OptionSelectedMsg:
		m.currentRoute = msg.Option
		// TODO Set new View
		return m, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		case msg.Type == tea.KeyCtrlC || msg.String() == "q":
			return m, tea.Quit
		default:
			m.mainMenu, cmd = m.mainMenu.Update(msg)
			if cmd != nil {
				return m, cmd
			}
			m.help, cmd = m.help.Update(msg)
		}
		return m, cmd
	default:
		var cmd tea.Cmd
		m.lblSpinner, cmd = m.lblSpinner.Update(msg)
		return m, cmd
	}
}

func (m *Model) View() string {
	var v strings.Builder
	v.WriteString(" 🛰️ ")
	if m.lblSpinner.IsSpinning {
		v.WriteString(m.lblSpinner.View())
	} else {
		v.WriteString(styles["mir"].Render("Mir"))
	}

	if m.err != nil {
		v.WriteString(m.err.Error())
		v.WriteString("\n")
	} else {
		v.WriteString(m.topLbl)
		v.WriteString("\n")
	}

	v.WriteString(m.mainMenu.View())
	v.WriteString(m.currentRoute)

	v.WriteString("\n\n" + m.help.View())

	return v.String()
}

type keyMap map[string]key.Binding

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["up"], k["down"], k["enter"]},
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
	"enter": key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "select"),
	),
}
