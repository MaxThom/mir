package mir

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/distributed_lock"
	"github.com/nats-io/nats.go"
)

// Expose Mir specific functions and domain
// Publicly expose nats io connection for custom usage
type Mir struct {
	Bus          *nats.Conn
	ctx          context.Context
	cancelFn     context.CancelFunc
	wg           *sync.WaitGroup
	name         string
	instanceName string
	LockStore    distributed_lock.DistributedLockStore
}

// MirStream is a stream that can be subscribed to
type MirStream interface {
	subject() string
	handler() nats.MsgHandler
}

// MirRequest is a request that can be sent to the Mir server
// and will return a reply
type MirRequest interface {
	msg() (*nats.Msg, error)
	response(*nats.Msg) error
}

// Establish connection to the Mir server
// This will enable communication to and from the device
// To properly close the connection, call the Close() function
func Connect(name string, target string) (*Mir, error) {
	m := &Mir{
		wg:           &sync.WaitGroup{},
		name:         name,
		instanceName: nats.NewInbox()[7:],
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
	m.LockStore, err = distributed_lock.NewNatsLockStore(m.Bus, m.instanceName)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Subscribe to a stream or event as a worker queue, this will put the handler
// in a go routine and listen for messages on that topic
func (m *Mir) QueueSubscribe(queue string, s ...MirStream) error {
	var errs error
	for _, stream := range s {
		sub, err := m.Bus.QueueSubscribeSync(stream.subject(), queue)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		m.wg.Add(1)
		go func(sub *nats.Subscription, handler nats.MsgHandler) {
			listenForStream(m.ctx, sub, handler)
			m.wg.Done()
		}(sub, stream.handler())
	}
	return errs
}

// Subscribe to a stream or event as a worker queue, this will put the handler
// in a go routine and listen for messages on that topic
func (m *Mir) QueueSubscribeRaw(subject string, queue string, handler nats.MsgHandler) error {
	var errs error
	sub, err := m.Bus.QueueSubscribeSync(subject, queue)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	m.wg.Add(1)
	go func(sub *nats.Subscription, handler nats.MsgHandler) {
		listenForStream(m.ctx, sub, handler)
		m.wg.Done()
	}(sub, handler)
	return errs
}

// Subscribe to a stream or event, this will put the handler
// in a go routine and listen for messages on that topic
func (m *Mir) Subscribe(s ...MirStream) error {
	var errs error
	for _, stream := range s {
		sub, err := m.Bus.SubscribeSync(stream.subject())
		if err != nil {
			errs = errors.Join(errs, err)
		}
		m.wg.Add(1)
		go func(sub *nats.Subscription, handler nats.MsgHandler) {
			listenForStream(m.ctx, sub, handler)
			m.wg.Done()
		}(sub, stream.handler())
	}
	return errs
}

// Subscribe to a stream or event, this will put the handler
// in a go routine and listen for messages on that topic
func (m *Mir) SubscribeRaw(subject string, handler nats.MsgHandler) error {
	var errs error
	sub, err := m.Bus.SubscribeSync(subject)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	m.wg.Add(1)
	go func(sub *nats.Subscription, handler nats.MsgHandler) {
		listenForStream(m.ctx, sub, handler)
		m.wg.Done()
	}(sub, handler)
	return errs
}

func (m Mir) GetInstanceName() string {
	return fmt.Sprint(m.name, "-", m.instanceName)
}

// Send a request to the Mir server
// and expect a reply
func (m *Mir) SendRequest(r MirRequest) error {
	msg, err := r.msg()
	msg.Header.Add("instance", m.GetInstanceName())
	if err != nil {
		return err
	}
	resp, err := m.Bus.RequestMsg(msg, 7*time.Second)
	if err != nil {
		return err
	}
	return r.response(resp)
}

func (m *Mir) SendRequestWithTimeout(r MirRequest, t time.Duration) error {
	msg, err := r.msg()
	msg.Header.Add("instance", m.GetInstanceName())
	if err != nil {
		return err
	}
	resp, err := m.Bus.RequestMsg(msg, t)
	if err != nil {
		return err
	}
	return r.response(resp)
}

// Listen the stream and call the handler
func listenForStream(ctx context.Context, sub *nats.Subscription, handler nats.MsgHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// TODO handle error
			msg, _ := sub.NextMsgWithContext(ctx)
			if msg != nil {
				handler(msg)
			}
		}
	}
}

// Close the connection to the Mir server
// Release all resources and stop all go routines
func (m *Mir) Disconnect() error {
	m.wg.Add(1)
	err := m.Bus.Drain()
	m.cancelFn()
	m.wg.Wait()
	return err
}
