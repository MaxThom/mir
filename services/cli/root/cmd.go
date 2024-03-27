package root

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
)

type (
	resMsg        = string
	connMirCmdMsg struct{}
	errMsg        struct{ err error }
)

func (e errMsg) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return ""
}

//func (m Model) Tick() tea.Msg {
//return m.spinner.Tick()
//}

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
			return errMsg{err}
		}
		m.bus = b

		time.Sleep(2 * time.Second)
		return resMsg("Mir")
	}
}
