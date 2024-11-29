package mirv2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/externals/distributed_lock"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
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

type Msg struct {
	*nats.Msg
}

func (m Msg) GetOriginalTriggerId() string {
	return m.Header.Get(HeaderOriginalTrigger)
}

const (
	HeaderRequestEnconding = "request-encoding"
	HeaderContentEncoding  = "content-encoding"
	HeaderOriginalTrigger  = "original-trigger"
	HeaderZstdEncoding     = "zstd"
)

// Establish connection to the Mir server
// This will enable communication to and from the device
// To properly close the connection, call the Close() function
func Connect(name string, target string) (*Mir, error) {
	m := &Mir{
		wg:           &sync.WaitGroup{},
		name:         name,
		instanceName: nats.NewInbox()[7:14],
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

// Close the connection to the Mir server
// Release all resources and stop all go routines
func (m *Mir) Disconnect() error {
	err := m.Bus.Drain()
	m.cancelFn()
	m.wg.Wait()
	return err
}

func (m Mir) GetInstanceName() string {
	return fmt.Sprint(m.name, "-", m.instanceName)
}

func (m *Mir) subscribe(subject string, h nats.MsgHandler) error {
	sub, err := m.Bus.Subscribe(subject, h)
	if err != nil {
		return err
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-m.ctx.Done()
		sub.Drain()
		sub.Unsubscribe()
	}()

	return nil
}

func (m *Mir) queueSubscribe(name, subject string, h nats.MsgHandler) error {
	sub, err := m.Bus.QueueSubscribe(subject, name, h)
	if err != nil {
		return err
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-m.ctx.Done()
		sub.Drain()
		sub.Unsubscribe()
	}()

	return nil
}

func (m *Mir) publish(subject string, data []byte, headers nats.Header) error {
	if headers == nil {
		headers = nats.Header{}
	}
	msg := &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    data,
	}

	return m.Bus.PublishMsg(msg)
}

func (m *Mir) request(subject string, data []byte, headers nats.Header) (*nats.Msg, error) {
	if headers == nil {
		headers = nats.Header{}
	}
	msg := &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    data,
	}

	return m.Bus.RequestMsg(msg, 7*time.Second)
}

func (m *Mir) requestWithCompression(subject string, data []byte, headers nats.Header) (*nats.Msg, error) {
	if headers == nil {
		headers = nats.Header{}
	}
	headers.Add(HeaderRequestEnconding, HeaderZstdEncoding)

	resp, err := m.request(subject, data, headers)
	if err != nil {
		return nil, fmt.Errorf("error publishing request message: %w", err)
	}
	if resp.Header.Get(HeaderContentEncoding) == HeaderZstdEncoding {
		resp.Data, err = zstd.DecompressData(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("error decompressing request data: %w", err)
		}
	}
	return resp, nil
}

func (m *Mir) sendReplyOrAck(msg *nats.Msg, resp proto.Message) error {
	if msg.Reply != "" {
		bResp, err := proto.Marshal(resp)
		if err != nil {
			msg.Ack()
			return fmt.Errorf("error marshalling response: %w", err)
		}
		err = m.Bus.Publish(msg.Reply, bResp)
		if err != nil {
			msg.Ack()
			return fmt.Errorf("error publishing response: %w", err)
		}
	}
	msg.Ack()
	return nil
}

func (m Mir) newEventId() (string, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("error creating event id: %w", err)
	}
	return u.String(), nil
}
