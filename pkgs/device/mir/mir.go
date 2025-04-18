package mir

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Mir struct {
	cfg         Cfg
	b           *bus.BusConn
	store       *Store
	ctx         context.Context
	cancelFn    context.CancelFunc
	cleanLogger zerolog.Logger
	l           zerolog.Logger
	schema      *descriptorpb.FileDescriptorSet
	schemaReg   *protoregistry.Files
	cmdHandlers map[string]cmdHandlerValue
	cfgHandlers map[string]cfgHandlerValue
	initialized bool
}

type Cfg struct {
	DeviceId       string       `json:"deviceId" yaml:"deviceId" cfg:""`
	Target         string       `json:"target" yaml:"target"`
	LogLevel       string       `json:"logLevel" yaml:"logLevel"`
	NoSchemaOnBoot bool         `json:"noSchemaOnBoot" yaml:"noSchemaOnBoot"`
	Store          StoreOptions `json:"store" yaml:"store"`
}

type cmdHandlerValue struct {
	t reflect.Type
	h func(proto.Message) (proto.Message, error)
}

type cfgHandlerValue struct {
	t reflect.Type
	h []func(proto.Message)
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
// Upon Launch, the device will request its properties from the server
// and call all the registered properties handlers
func (m *Mir) Launch(ctx context.Context) (*sync.WaitGroup, error) {
	var err error
	var wg sync.WaitGroup
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx, m.cancelFn = context.WithCancel(ctx)
	connectWorkDone := make(chan struct{})

	// Setup persistence
	if err := m.store.Load(); err != nil {
		return &wg, fmt.Errorf("error loading local store: %w", err)
	}
	m.l.Debug().Msg("persistence loaded")

	// Setup Mir bus
	m.b, err = bus.New(m.cfg.Target,
		bus.WithConnectHandler(func(nc *nats.Conn) {
			m.l.Info().Msg("connected to Mir Server ")
			m.setOnlineHandler()

			if !m.cfg.NoSchemaOnBoot {
				if err := m.sendSchema(); err != nil {
					m.l.Error().Err(err).Msg("error sending schema on connect")
				}
				m.l.Debug().Msg("schema updated")
			}

			// Call config handler
			if err := m.requestDesiredProperties(); err != nil {
				m.l.Error().Err(err).Msg("error requesting desired properties")
			}

			go func() {
				time.Sleep(1 * time.Second)
				m.sendPendingMsgs()
			}()

			close(connectWorkDone)
		}),
		bus.WithReconnHandler(func(nc *nats.Conn) {
			m.l.Info().Msg("reconnected to Mir Server ")
			// This time gives time for server side service
			// to reconnect if they were disconnected as well
			time.Sleep(1 * time.Second)
			m.setOnlineHandler()
			if err := m.requestDesiredProperties(); err != nil {
				m.l.Error().Err(err).Msg("error requesting desired properties")
			}
			m.callAllCfgHandlers()
			m.l.Debug().Msg("desired properties propagated")

			go func() {
				time.Sleep(1 * time.Second)
				m.sendPendingMsgs()
			}()
		}),
		bus.WithDisconnHandler(func(_ *nats.Conn, err error) {
			m.l.Warn().Err(err).Msg("disconnected from Mir Server")
			m.setOfflineHandler()
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			m.l.Warn().Msg("closed connection from Mir Server")
		}),
		bus.WithReconnect())
	if err != nil {
		return &wg, err
	}

	sub, err := m.b.SubscribeSync(fmt.Sprintf("%s.>", m.cfg.DeviceId))
	if err != nil {
		return &wg, err
	}

	wg.Add(1)
	go func() {
		m.commands(ctx, sub)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		m.hearthbeat(m.ctx, time.Second*10)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		m.shutdown(m.ctx)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		<-ctx.Done()
		err := m.store.Close()
		if err != nil {
			m.l.Error().Err(err).Msg("error closing device store")
		}
		m.l.Info().Msg("flushing writes to store and closing")
		wg.Done()
	}()

	select {
	case <-connectWorkDone:
		m.l.Debug().Msg("online initialization")
	case <-time.After(2 * time.Second):
		m.l.Warn().Msg("disconnected from Mir Server ")
		m.setOfflineHandler()
		m.l.Debug().Msg("offline initialization")
	}

	m.callAllCfgHandlers()
	m.l.Debug().Msg("desired properties propagated")
	m.initialized = true
	m.l.Debug().Msg("device initialized")

	// Wait for the connection to be established
	// and device created on the server if needed

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
				err = msgHandlers[clients.DeviceSubject(msg.Subject).GetVersionAndFunction()](msg, m)
				if err != nil {
					m.l.Error().Err(err).Msg("error handling command " + clients.DeviceSubject(msg.Subject).GetVersionAndFunction())
				}
			}
		}
	}
}

func (m *Mir) shutdown(ctx context.Context) {
	// for {
	select {
	case <-ctx.Done():
		m.l.Info().Msg("shutting down connection to Mir")
		m.b.Conn.Close()
		return
	}
	// }
}

// Return a new context of the Mir SDK logger
// The zerolog.logger can be extended and
// used to log your app specific logs
// You need to assigned it first
// eg.
// l := m.Logger()
// l.Info().Msg("Mir is ready for launch")
func (m Mir) Logger() *zerolog.Logger {
	l := m.cleanLogger.With().Logger()
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
	msg, err := tlm_client.GetTelemetryStreamMsg(m.cfg.DeviceId, t)
	if err != nil {
		return err
	}
	return m.sendMsg(msg)
}

// Send proto reported properties to Mir Server
func (m Mir) SendProperties(t proto.Message) error {
	msg, err := cfg_client.GetReportedPropertiesStreamMsg(m.cfg.DeviceId, t)
	if err != nil {
		return err
	}
	return m.sendMsg(msg)
}

func (m Mir) sendSchema() error {
	bytes, err := proto.Marshal(m.schema)
	if err != nil {
		return nil
	}

	return m.sendProtoMsg(core_client.SchemaDeviceStream.WithId(m.GetDeviceId()), &device_apiv1.SchemaRetrieveResponse{
		Response: &device_apiv1.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, true)
}

// Fill the properties store with the latest properties from Mir server
// Also write to the persistent store
func (m Mir) requestDesiredProperties() error {
	resp, err := cfg_client.PublishRequestDesiredPropertiesStream(m.b, m.cfg.DeviceId)
	if err != nil {
		return fmt.Errorf("error requesting desired properties: %v", err)
	}
	if resp.GetError() != "" {
		return fmt.Errorf("error requesting desired properties: %v", resp.GetError())
	}

	props := resp.GetOk()

	var errs error
	for msgName, cfg := range props.Properties {
		updTime, err := time.Parse(time.RFC3339, cfg.Time)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		if _, err := m.store.UpdatePropsIfNew(msgName, propsValue{LastUpdate: updTime, Value: cfg.Property}); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}

	return errs
}

func (m Mir) callAllCfgHandlers() {
	for name, h := range m.cfgHandlers {
		props, ok := m.store.GetProps(name)
		if !ok {
			continue
		}

		v := reflect.New(h.t).Interface()
		msg := v.(proto.Message)
		if err := proto.Unmarshal(props.Value, msg); err != nil {
			m.l.Error().Err(err).Msg("error unmarshalling properties")
			continue
		}

		for _, handler := range h.h {
			handler(msg)
		}
	}
}

// Handle a command from Mir server
// Specify which command with the proto msg from your schema
// Return a proto message as response, nil if no response, or an error
// eg:
// m.HandleCommand(
//
//	&config_devicev1.SendConfigRequest{},
//	func(m proto.Message) (proto.Message, error) {
//	  cmd := m.(*config_devicev1.SendConfigRequest)
//	  return &config_devicev1.SendConfigResponse{}, nil
//	})
func (m Mir) HandleCommand(t proto.Message, handler func(proto.Message) (proto.Message, error)) {
	m.cmdHandlers[string(t.ProtoReflect().Descriptor().FullName())] = cmdHandlerValue{
		t: reflect.TypeOf(t).Elem(),
		h: handler,
	}
}

// Handle a properties update from Mir server
// Specify which properties with the proto msg from your schema
// Each properties can have many handlers and each will be called upon update
// If the handler is registered after the launch, it will be called if the properties
// is already present in the store
// If the handler is registered before the launch, it will be called on Launch
// eg:
// m.HandleProperties(
//
//	&config_devicev1.SendConfigRequest{},
//	func(m proto.Message) {
//	  cmd := m.(*config_devicev1.SendConfigRequest)
//	})
func (m Mir) HandleProperties(t proto.Message, handler ...func(proto.Message)) {
	key := string(t.ProtoReflect().Descriptor().FullName())
	cfg, ok := m.cfgHandlers[key]
	if !ok {
		cfg = cfgHandlerValue{
			t: reflect.TypeOf(t).Elem(),
			h: []func(proto.Message){},
		}
	}
	cfg.h = append(cfg.h, handler...)
	m.cfgHandlers[key] = cfg

	// If we register cfg handler after launch, we need to call it
	// if we already have the properties
	if m.initialized {
		props, ok := m.store.GetProps(key)
		if !ok {
			return
		}

		v := reflect.New(cfg.t).Interface()
		msg := v.(proto.Message)
		if err := proto.Unmarshal(props.Value, msg); err != nil {
			m.l.Error().Err(err).Msg("error unmarshalling properties")
			return
		}

		for _, handler := range handler {
			handler(msg)
		}
	}
}

type subject string

func (m Mir) NewSubject(module, version, function string, extra ...string) subject {
	extra = append([]string{"device", m.cfg.DeviceId, module, version, function}, extra...)
	return subject(strings.Join(extra, "."))
}

// Send custom data on a custom route for your own integration
// use `m.NewSubject` to create a subject
// Use the module sdk to subscribe to the subject and process the data
func (m Mir) SendData(sbj subject, data []byte, h nats.Header) error {
	return m.sendMsg(&nats.Msg{
		Subject: string(sbj),
		Header:  h,
		Data:    data,
	})
}
