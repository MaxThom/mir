package mir

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
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
}

type Msg struct {
	*nats.Msg
}

func NewMsg(subject string) *Msg {
	return &Msg{
		Msg: nats.NewMsg(subject),
	}
}

// Mainly present for Events
// Represent the serverId that made the request trigger
func (m Msg) GetOriginalTriggerId() string {
	return m.Header.Get(HeaderPreviousTrigger)
}

// ServerId that triggered the event
func (m Msg) GetOrigin() string {
	return m.Header.Get(HeaderTrigger)
}

func (m *Msg) AddToTriggerChain(s ...string) {
	if m.Header == nil {
		m.Header = nats.Header{}
	}
	for _, v := range s {
		m.Header.Add(HeaderTrigger, v)
	}
}

func (m Msg) GetTriggerChain() []string {
	return m.Header.Values(HeaderTrigger)
}

func (m Msg) GetProtoMsgName() string {
	return m.Header.Get(HeaderMsgName)
}

func (m *Msg) SetProtoMsgName(b string) {
	if m.Header == nil {
		m.Header = nats.Header{}
	}
	m.Header.Set(HeaderMsgName, b)
}

func (m *Msg) GetTime() time.Time {
	t, err := time.Parse(time.RFC3339Nano, m.Header.Get(HeaderTime))
	if err != nil {
		return time.Time{}
	}
	return t
}

func (m *Msg) SetTime(t time.Time) {
	if m.Header == nil {
		m.Header = nats.Header{}
	}
	m.Header.Set(HeaderTime, t.Format(time.RFC3339Nano))
}

const (
	HeaderRequestEnconding = "mir-request-encoding"
	HeaderContentEncoding  = "mir-content-encoding"
	HeaderPreviousTrigger  = "mir-previous-trigger"
	HeaderTrigger          = "mir-trigger-chain"
	HeaderRoute            = "mir-route"
	HeaderSubject          = "mir-subject"
	HeaderZstdEncoding     = "mir-zstd"
	HeaderMsgName          = "mir-msg"
	HeaderTime             = "mir-time"
)

// Establish connection to the Mir server
// This will enable communication to and from the device
// To properly close the connection, call the Close() function
func Connect(name string, target string, natsOpts ...nats.Option) (*Mir, error) {
	m := &Mir{
		wg:           &sync.WaitGroup{},
		name:         name,
		instanceName: nats.NewInbox()[7:14],
	}
	m.ctx, m.cancelFn = context.WithCancel(context.Background())

	// Setup Mir bus
	var err error
	m.Bus, err = nats.Connect(target,
		append([]nats.Option{nats.Name(name)}, natsOpts...)...,
	)
	if err != nil {
		return m, err
	}

	return m, nil
}

func WithDefaultReconnectOpts() []nats.Option {
	return []nats.Option{
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(3 * time.Second),
		nats.ReconnectJitter(time.Millisecond*100, time.Second), // Set the jitter for reconnects
	}
}

func WithDefaultConnectionLogging(l zerolog.Logger) []nats.Option {
	return []nats.Option{
		nats.ConnectHandler(func(c *nats.Conn) {
			l.Info().Msg("connected to Mir Server ")
		}),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			l.Warn().Err(err).Msg("disconnected from Mir Server")
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			l.Info().Msg("reconnected to Mir Server ")
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			l.Warn().Msg("closed connection from Mir Server")
		}),
	}
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
	headers.Add(HeaderTrigger, m.GetInstanceName())

	msg := &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    data,
	}
	return m.Bus.PublishMsg(msg)
}

func (m *Mir) request(subject string, data []byte, headers nats.Header, timeout time.Duration) (*nats.Msg, error) {
	if headers == nil {
		headers = nats.Header{}
	}
	headers.Add(HeaderTrigger, m.GetInstanceName())

	msg := &nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    data,
	}
	return m.Bus.RequestMsg(msg, timeout)
}

func (m *Mir) decompressMsg(msg *nats.Msg) (*nats.Msg, error) {
	if msg.Header.Get(HeaderContentEncoding) == HeaderZstdEncoding {
		var err error
		msg.Data, err = zstd.DecompressData(msg.Data)
		if err != nil {
			return nil, fmt.Errorf("error decompressing request data: %w", err)
		}
	}
	return msg, nil
}

func (m *Mir) requestWithCompression(subject string, data []byte, headers nats.Header, timeout time.Duration) (*nats.Msg, error) {
	if headers == nil {
		headers = nats.Header{}
	}
	headers.Add(HeaderRequestEnconding, HeaderZstdEncoding)

	resp, err := m.request(subject, data, headers, timeout)
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
		headers := nats.Header{}
		headers.Add(HeaderTrigger, m.GetInstanceName())
		reply := &nats.Msg{
			Subject: msg.Reply,
			Header:  headers,
			Data:    bResp,
		}
		err = m.Bus.PublishMsg(reply)
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
