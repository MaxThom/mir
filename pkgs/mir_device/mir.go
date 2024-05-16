package mir_device

import (
	"context"
	"fmt"
	"sync"
	"time"

	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/core"
	"github.com/nats-io/nats.go"
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
}

type cfg struct {
	DeviceId string `json:"deviceId" yaml:"deviceId"`
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
			//logger.Warn().Msg("reconnected to " + nc.ConnectedUrl())
		}),
		bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
			//logger.Warn().Msg(fmt.Sprintf("disconnected due to %v, will attempt to reconnect ", err))
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			//logger.Warn().Msg("connection to %v closed " + nc.ConnectedUrl())
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

	fmt.Println(m.cfg.DeviceId)
	fmt.Println(m.cfg.Target)
	fmt.Println(m.b.ConnectedServerName())

	return &wg, nil
}

// Send hearthbeat to Mir on a based intervall
// Run in a routine for non blocking
func (m *Mir) hearthbeat(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("HEARTHBEAT CLOSING")
			return
		case <-time.After(interval):
			if err := core.PublishHearthbeatRequest(m.b, m.cfg.DeviceId); err != nil {
				// log the error
			}
		}
	}
}

func (m *Mir) shutdown(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("CLOSING")
			m.b.Conn.Close()
			return
		}
	}
}
