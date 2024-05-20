package mir_device

import (
	"context"
	"fmt"
	"sync"
	"time"

	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/core"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// IDEA on go language
// have return variable at outside scope sugar syntax
// bus, db if err := ConnectToSystems(bUrl string, dbUrl string); err != nil {
// 	err is in scope here as well as bus and db
// } else {
//  same here
// }
// bus and db still exist here, but not err

type Mir struct {
	cfg      cfg
	b        *bus.BusConn
	ctx      context.Context
	cancelFn context.CancelFunc
	l        zerolog.Logger
}

type cfg struct {
	DeviceId string `json:"deviceId" yaml:"deviceId" cfg:""`
	Target   string `json:"target" yaml:"target"`
	LogLevel string `json:"logLevel" yaml:"logLevel"`
}

const ()

var ()

// Establish connection to the Mir server
// This will enable communication to and from the device
// For a gracefull shutdown, simply wait the returning waitgroup after
// cancelling the context
func (m *Mir) Launch(ctx context.Context) (*sync.WaitGroup, error) {
	var wg sync.WaitGroup
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx, m.cancelFn = context.WithCancel(ctx)

	// Setup Mir bus
	var err error
	m.b, err = bus.New(m.cfg.Target,
		bus.WithReconnHandler(func(nc *nats.Conn) {
			m.l.Warn().Msg("reconnected to Mir Server ")
		}),
		bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				m.l.Error().Err(err).Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
			}
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
		}))
	if err != nil {
		return &wg, err
	}

	go func() {
		wg.Add(1)
		m.hearthbeat(m.ctx, time.Second*10)
		wg.Done()
	}()

	go func() {
		wg.Add(1)
		m.shutdown(m.ctx)
		wg.Done()
	}()

	return &wg, nil
}

// Send hearthbeat to Mir on a based intervall
// Run in a routine for non blocking
func (m *Mir) hearthbeat(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			m.l.Debug().Msg("shuting down hearthbeat")
			return
		case <-time.After(interval):
			if err := core.PublishHearthbeatRequest(m.b, m.cfg.DeviceId); err != nil {
				m.l.Error().Err(err).Msg("error sending hearthbeat to Mir")
			}
		}
	}
}

func (m *Mir) shutdown(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			m.l.Info().Msg("shutting down connection to Mir")
			m.b.Conn.Close()
			return
		}
	}
}

// Return a new context of the Mir SDK logger
// The zerolog.logger can be extended and
// used to log your app specific logs
// You need to assigned it first
// eg.
// l := m.Logger()
// l.Info().Msg("Mir is ready for launch")
func (m Mir) Logger() zerolog.Logger {
	l := m.l.With().Logger()
	return l
}
