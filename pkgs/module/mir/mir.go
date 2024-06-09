package mir

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Mir struct {
	Bus      *nats.Conn
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       *sync.WaitGroup
}

type MirStream interface {
	subject() string
	handler() nats.MsgHandler
}

type MirRequest interface {
	msg() (*nats.Msg, error)
	response(*nats.Msg) error
}

// Establish connection to the Mir server
// This will enable communication to and from the device
// For a gracefull shutdown, simply wait the returning waitgroup after
// cancelling the context
func Connect(name string, target string) (*Mir, error) {
	m := &Mir{
		wg: &sync.WaitGroup{},
	}
	m.ctx, m.cancelFn = context.WithCancel(context.Background())

	// Setup Mir bus
	var err error
	m.Bus, err = nats.Connect(target,
		[]nats.Option{
			nats.Name(name),
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(3 * time.Second),
			nats.DisconnectErrHandler(func(c *nats.Conn, err error) {

			}),
			nats.ReconnectHandler(func(c *nats.Conn) {

			}),
			nats.ClosedHandler(func(c *nats.Conn) {
				m.wg.Done()
			}),
		}...,
	)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Mir) Subscribe(s ...MirStream) error {
	var errs error
	for _, stream := range s {
		sub, err := m.Bus.SubscribeSync(stream.subject())
		if err != nil {
			errs = errors.Join(errs, err)
		}
		go func(sub *nats.Subscription, handler nats.MsgHandler) {
			m.wg.Add(1)
			listenForStream(m.ctx, sub, handler)
			m.wg.Done()
		}(sub, stream.handler())
	}
	return errs
}

func (m *Mir) SendRequest(r MirRequest) error {
	msg, err := r.msg()
	if err != nil {
		return err
	}
	resp, err := m.Bus.RequestMsg(msg, 7*time.Second)
	if err != nil {
		return err
	}
	return r.response(resp)
}

func listenForStream(ctx context.Context, sub *nats.Subscription, handler nats.MsgHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// TODO handle error
			msg, _ := sub.NextMsgWithContext(ctx)
			handler(msg)
		}
	}
}

func (m *Mir) Disconnect() error {
	m.wg.Add(1)
	err := m.Bus.Drain()
	m.cancelFn()
	m.wg.Wait()
	return err
}
