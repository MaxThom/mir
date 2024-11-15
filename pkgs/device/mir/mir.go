package mir

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
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
	cfg         Cfg
	b           *bus.BusConn
	ctx         context.Context
	cancelFn    context.CancelFunc
	l           zerolog.Logger
	schema      *descriptorpb.FileDescriptorSet
	schemaReg   *protoregistry.Files
	cmdHandlers map[string]cmdHandlerValue
}

type Cfg struct {
	DeviceId string `json:"deviceId" yaml:"deviceId" cfg:""`
	Target   string `json:"target" yaml:"target"`
	LogLevel string `json:"logLevel" yaml:"logLevel"`
}

type cmdHandlerValue struct {
	t reflect.Type
	h func(proto.Message) (proto.Message, error)
}

const ()

var ()

func (m Mir) GetConfig() Cfg {
	return m.cfg
}

func (m Mir) GetDeviceId() string {
	return m.cfg.DeviceId
}

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

	sub, err := m.b.SubscribeSync(fmt.Sprintf("%s.>", m.cfg.DeviceId))
	if err != nil {
		return &wg, err
	}
	go func() {
		wg.Add(1)
		m.commands(ctx, sub)
		wg.Done()
	}()

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

// Send hearthbeat to Mir on a based interval
// Run in a routine for non blocking
func (m *Mir) hearthbeat(ctx context.Context, interval time.Duration) {
	if err := core_client.PublishHearthbeatStream(m.b, m.cfg.DeviceId); err != nil {
		m.l.Error().Err(err).Msg("error sending hearthbeat to Mir")
	}
	for {
		select {
		case <-ctx.Done():
			m.l.Debug().Msg("shutting down hearthbeat")
			return
		case <-time.After(interval):
			if err := core_client.PublishHearthbeatStream(m.b, m.cfg.DeviceId); err != nil {
				m.l.Error().Err(err).Msg("error sending hearthbeat to Mir")
			}
		}
	}
}

// Listen to any incoming commands from Mir and redirect to handler
// Run in a routine for non blocking
func (m *Mir) commands(ctx context.Context, sub *nats.Subscription) {
	for {
		select {
		case <-ctx.Done():
			m.l.Debug().Msg("shutting down command listener")
			return
		default:
			// TODO error channel
			msg, err := sub.NextMsgWithContext(ctx)
			if err != nil && !strings.Contains(err.Error(), "context canceled") { // TODO if not context canceled
				m.l.Error().Err(err).Msg("error receiving commands")
			}
			if msg != nil {
				m.l.Debug().Msg("handling command " + clients.DeviceSubject(msg.Subject).GetVersionAndFunction())
				err = cmdHandlers[clients.DeviceSubject(msg.Subject).GetVersionAndFunction()](msg, m)
				if err != nil {
					m.l.Error().Err(err).Msg("error handling command " + clients.DeviceSubject(msg.Subject).GetVersionAndFunction())
				}
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
func (m Mir) Logger() *zerolog.Logger {
	l := m.l.With().Logger()
	return &l
}

// Marshal the FileDescriptorSet to bytes
func (m Mir) marshalTelemetrySchema() ([]byte, error) {
	var schemaBytes []byte
	var err error
	if m.schema != nil {
		schemaBytes, err = proto.Marshal(m.schema)
	}
	return schemaBytes, err
}

// Send proto telemetry to Mir Server
func (m Mir) SendTelemetry(t proto.Message) error {
	return tlm_client.PublishTelemetryStream(m.b, m.cfg.DeviceId, t)
}

func (m Mir) HandleCommand(t proto.Message, handler func(proto.Message) (proto.Message, error)) {
	m.cmdHandlers[string(t.ProtoReflect().Descriptor().FullName())] = cmdHandlerValue{
		t: reflect.TypeOf(t).Elem(),
		h: handler,
	}
}
