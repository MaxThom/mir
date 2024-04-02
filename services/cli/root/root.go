package root

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/cli/components/labelspinner"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

type Model struct {
	ctx        context.Context
	bus        *bus.BusConn
	mirUrl     string
	topLbl     string
	err        error
	lblSpinner labelspinner.Model
	isSpinning bool
}

func NewModel(ctx context.Context, log zerolog.Logger, mirUrl string) *Model {
	l = log.With().Str("cmp", "root").Logger()
	s := labelspinner.New("", spinner.Dot)
	return &Model{
		ctx:        ctx,
		mirUrl:     mirUrl,
		lblSpinner: s,
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
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	default:
		var cmd tea.Cmd
		m.lblSpinner, cmd = m.lblSpinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	// "🛰️⣿"
	msg := " 🛰️ "
	if m.lblSpinner.IsSpinning {
		msg += m.lblSpinner.View()
	} else {
		msg += "Mir"
	}

	if m.err != nil {
		msg += "" + m.err.Error() + "\n"
	} else {
		msg += "" + m.topLbl + "\n"
	}
	return msg
}
