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
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/device_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type (
	Mir struct {
		cfg         Config
		b           *bus.BusConn
		store       *Store
		ctx         context.Context
		cancelFn    context.CancelFunc
		cleanLogger zerolog.Logger
		l           zerolog.Logger
		schema      *descriptorpb.FileDescriptorSet
		schemaReg   *protoregistry.Files
		msgHandlers map[string]msgHandler
		cmdHandlers map[string]cmdHandlerValue
		cfgHandlers map[string]cfgHandlerValue
		msgSender   msgSender
		initialized bool
	}

	Config struct {
		Target      string       `json:"target" yaml:"target"`
		Credentials string       `json:"credentials" yaml:"credentials"`
		RootCA      string       `json:"rootCA" yaml:"rootCA"`
		TLSCert     string       `json:"tlsCert" yaml:"tlsCert"`
		TLSKey      string       `json:"tlsKey" yaml:"tlsKey"`
		LogLevel    string       `json:"logLevel" yaml:"logLevel"`
		Device      DeviceCfg    `json:"device" yaml:"device"`
		LocalStore  StoreOptions `json:"localStore" yaml:"localStore"`
	}

	DeviceCfg struct {
		Id             string       `json:"id" yaml:"id"`
		IdPrefix       *IdPrefix    `json:"idPrefix,omitempty" yaml:"idPrefix"`
		IdGenerator    *IdGenerator `json:"idGenerator,omitempty" yaml:"idGenerator"`
		NoSchemaOnBoot bool         `json:"noSchemaOnBoot" yaml:"noSchemaOnBoot"`
	}

	msgHandler func(msg *nats.Msg) error
	msgSender  func(msg *nats.Msg) error

	cmdHandlerValue struct {
		t reflect.Type
		h func(proto.Message) (proto.Message, error)
	}

	cfgHandlerValue struct {
		t reflect.Type
		h []func(proto.Message)
	}
)

// Launch establish connection to the Mir server.
//   - This will enable communication to and from the device.
//   - For a gracefull shutdown, simply wait the returning waitgroup after cancelling the context
//   - Upon Launch, the device will request its properties from the server and call all the registered properties handlers.
//   - Pass extra Nats options for more control and options.
func (m *Mir) Launch(ctx context.Context, opts ...nats.Option) (*sync.WaitGroup, error) {
	var err error
	var wg sync.WaitGroup
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx, m.cancelFn = context.WithCancel(ctx)
	connectWorkDone := make(chan struct{})

	// Setup msg handlers
	// Custom messages from server that the SDK handles
	m.msgHandlers[device_client.SchemaRequest.GetVersionAndFunction()] = m.schemaRetrieveHandler
	m.msgHandlers[device_client.CommandRequest.GetVersionAndFunction()] = m.definedCommandHandler
	m.msgHandlers[device_client.ConfigRequest.GetVersionAndFunction()] = m.definedConfigHandler

	// Setup persistence
	if err := m.store.Load(); err != nil {
		return &wg, fmt.Errorf("error loading local store: %w", err)
	}
	m.l.Debug().Msg("persistence loaded")

	// Setup Mir bus
	m.b, err = bus.New(m.cfg.Target,
		bus.WithUserCredentials(m.cfg.Credentials),
		bus.WithRootCA(m.cfg.RootCA),
		bus.WithClientCertificate(m.cfg.TLSCert, m.cfg.TLSKey),
		bus.WithCustom(opts...),
		bus.WithConnectHandler(func(nc *nats.Conn) {
			m.l.Info().Msg("connected to Mir Server ")
			m.setOnlineHandler()

			if !m.cfg.Device.NoSchemaOnBoot {
				if err := core_client.PublishHearthbeatWithHello(m.b.Conn, m.cfg.Device.Id, m.schema); err != nil {
					m.l.Error().Err(err).Msg("error sending initial hello hearthbeat to Mir")
				}
			} else {
				if err := core_client.PublishHearthbeatStream(m.b.Conn, m.cfg.Device.Id); err != nil {
					m.l.Error().Err(err).Msg("error sending initial hearthbeat to Mir")
				}
			}
			m.l.Debug().Bool("with_schema", m.cfg.Device.NoSchemaOnBoot).Msg("initial hello hearthbeat sent")

			time.Sleep(1 * time.Second)

			// Call config handler
			if err := m.requestDesiredProperties(); err != nil {
				if strings.Contains(err.Error(), "no device found with current targets criteria") {
					m.l.Warn().Msg("device not found on Mir Server, auto commissionning a new device")
				} else {
					m.l.Error().Err(err).Msg("error requesting desired properties, using local")
				}
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
			// TODO loop over here on error instead of sleep
			time.Sleep(1 * time.Second)
			m.setOnlineHandler()
			if err := m.requestDesiredProperties(); err != nil {
				m.l.Error().Err(err).Msg("error requesting desired properties, using local")
			}
			m.callAllCfgHandlers()
			m.l.Debug().Msg("desired properties propagated")

			go func() {
				time.Sleep(1 * time.Second)
				m.sendPendingMsgs()
			}()
		}),
		bus.WithDisconnHandler(func(c *nats.Conn, err error) {
			m.l.Warn().Err(err).Err(c.LastError()).Msg("disconnected from Mir Server")
			m.setOfflineHandler()
		}),
		bus.WithClosedHandler(func(nc *nats.Conn) {
			m.l.Warn().Err(nc.LastError()).Msg("closed connection from Mir Server")
		}),
		bus.WithReconnect())
	if err != nil {
		return &wg, err
	}

	sub, err := m.b.SubscribeSync(fmt.Sprintf("%s.>", m.cfg.Device.Id))
	if err != nil {
		return &wg, err
	}

	wg.Go(func() {
		m.hearthbeat(m.ctx, time.Second*10)
	})

	wg.Go(func() {
		m.msgHandlerSub(ctx, sub)
	})

	wg.Go(func() {
		<-ctx.Done()
		m.shutdown()
	})

	wg.Go(func() {
		<-ctx.Done()
		err := m.store.Close()
		if err != nil {
			m.l.Error().Err(err).Msg("error closing device store")
		}
		m.l.Debug().Msg("flushed writes to store")
	})

	// This timeout has to be longer then the connect handler possible time to take
	select {
	case <-connectWorkDone:
		m.l.Debug().Msg("online initialization")
	case <-time.After(40 * time.Second):
		m.l.Warn().Str("connection_status", m.b.Status().String()).Err(m.b.LastError()).Msg("graceful initialization time out")
		if m.b.Status() == nats.CONNECTED {
			m.setOnlineHandler()
			m.l.Debug().Msg("partial online initialization")
		} else {
			m.l.Warn().Err(m.b.LastError()).Msg("disconnected from Mir Server ")
			m.setOfflineHandler()
			m.l.Debug().Msg("offline initialization")
		}
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
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			m.l.Debug().Msg("shutting down hearthbeat")
			return
		case <-timer.C:
			if m.b.Status() == nats.CONNECTED {
				if err := core_client.PublishHearthbeatStream(m.b.Conn, m.cfg.Device.Id); err != nil {
					m.l.Error().Err(err).Msg("error sending hearthbeat to Mir")
				}
			}
		}
	}
}

// Listen to any incoming msg from Mir and redirect to handler
// Run in a routine for non blocking
func (m *Mir) msgHandlerSub(ctx context.Context, sub *nats.Subscription) {
	for {
		select {
		case <-ctx.Done():
			m.l.Debug().Msg("shutting down command listener")
			return
		default:
			// TODO error channel
			msg, err := sub.NextMsgWithContext(ctx)
			if err != nil && !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "connection closed") { // TODO if not context canceled
				m.l.Error().Err(err).Msg("error receiving message")
			}
			if msg != nil {
				// TODO manage msg in a go routine so its not blocking
				go func() {
					m.l.Debug().Str("subject", clients.DeviceSubject(msg.Subject).GetVersionAndFunction()).Str("msg", msg.Header.Get(HeaderMsgName)).Msg("handling message")
					err = m.msgHandlers[clients.DeviceSubject(msg.Subject).GetVersionAndFunction()](msg)
					if err != nil {
						m.l.Error().Err(err).Str("subject", clients.DeviceSubject(msg.Subject).GetVersionAndFunction()).Str("msg", msg.Header.Get(HeaderMsgName)).Msg("error handling message")
					}
				}()
			}
		}
	}
}

func (m *Mir) shutdown() {
	m.l.Info().Msg("shutting down connection to Mir")
	m.b.Conn.Close()
}

// Call all cfg handlers with the properties from the store
func (m *Mir) callAllCfgHandlers() {
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
			go handler(msg)
		}
	}
}

// Select message handler according to online state and cfg
func (m *Mir) setOnlineHandler() {
	switch m.store.opts.PersistenceType {
	case PersistentTypeNoStorage:
		m.msgSender = m.sendMsgOnly
		m.l.Info().Msg("set online handler: no storage")
	case PersistentTypeOnlyIfOffline:
		m.msgSender = m.sendMsgOnly
		m.l.Info().Msg("set online handler: no storage")
	case PersistentTypeAlways:
		m.msgSender = m.sendMsgWithStorage
		m.l.Info().Msg("set online handler: persistent storage")
	}
}

// Select message handler according to offline state and cfg
func (m *Mir) setOfflineHandler() {
	switch m.store.opts.PersistenceType {
	case PersistentTypeNoStorage:
		m.msgSender = m.sendNothing
		m.l.Info().Msg("set offline handler: no storage")
	case PersistentTypeOnlyIfOffline:
		m.msgSender = m.saveMsgInPending
		m.l.Info().Msg("set offline handler: pending storage")
	case PersistentTypeAlways:
		m.msgSender = m.saveMsgInPending
		m.l.Info().Msg("set offline handler: pending storage")
	}
}

func (m Mir) sendMsg(msg *nats.Msg) error {
	return m.msgSender(msg)
}

// Discard any msg to be sent.
// Cause: offline and no storage
func (m Mir) sendNothing(msg *nats.Msg) error {
	return nil
}

// Send message to server
// Cause: online and no storaage
func (m Mir) sendMsgOnly(msg *nats.Msg) error {
	return m.b.PublishMsg(msg)
}

// Save msg to pending store
// Cause: offline and save if offline
func (m Mir) saveMsgInPending(msg *nats.Msg) error {
	return m.store.SaveMsgToPending(*msg)
}

// Save msg to permenant storage
// Cause: online and persistent cfg
// Performance solution would be to do batch writes
// and split DISK IO and Internet IO
func (m Mir) sendMsgWithStorage(msg *nats.Msg) error {
	if err := m.store.SaveMsgToPermanent(*msg); err != nil {
		m.l.Warn().Err(err).Msg("error saving msg to sent store")
	}
	return m.b.PublishMsg(msg)
}

func (m Mir) sendPendingMsgs() {
	batchSize := 100
	if m.cfg.LocalStore.PersistenceType == PersistentTypeAlways {
		count := 0
		if err := m.store.SwapMsgByBatch(msgPendingBucket, msgPersistentBucket, batchSize, func(msgs []nats.Msg) error {
			var errs error
			for _, msg := range msgs {
				errs = errors.Join(m.sendMsgOnly(&msg))
			}
			count += len(msgs)
			return errs
		}); err != nil {
			m.l.Error().Err(err).Msg("error sending pending messages to Mir")
		}
		if count == 0 {
			m.l.Info().Msg("pending storage is empty")
		} else {
			m.l.Info().Msgf("%d pending messages sent to Mir and moved to persistent storage", count)
		}
	} else {
		count := 0
		if err := m.store.DeleteMsgByBatch(msgPendingBucket, batchSize, func(msgs []nats.Msg) error {
			var errs error
			for _, msg := range msgs {
				errs = errors.Join(m.sendMsgOnly(&msg))
			}
			count += len(msgs)
			return errs
		}); err != nil {
			m.l.Error().Err(err).Msg("error sending pending messages to Mir")
		}
		if count == 0 {
			m.l.Info().Msg("pending storage is empty")
		} else {
			m.l.Info().Msgf("%d pending messages sent to Mir", count)
		}
	}
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

func (m Mir) sendSchema() error {
	bytes, err := proto.Marshal(m.schema)
	if err != nil {
		return err
	}

	return m.sendProtoMsg(core_client.SchemaDeviceStream.WithId(m.GetDeviceId()), &mir_apiv1.SchemaRetrieveResponse{
		Response: &mir_apiv1.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, true)
}
