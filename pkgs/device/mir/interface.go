package mir

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Return a new context of the Mir SDK logger
// The zerolog.logger can be extended and
// used to log your app specific logs
// You need to assigned it first
// eg.
// l := m.Logger()
// l.Info().Msg("Mir is ready for launch")
func (m *Mir) Logger() *zerolog.Logger {
	l := m.cleanLogger.With().Logger()
	return &l
}

// Return device configuration
func (m Mir) GetConfig() Config {
	return m.cfg
}

// Return device id used for identification
func (m Mir) GetDeviceId() string {
	return m.cfg.Device.Id
}

// Send proto telemetry to Mir Server
func (m *Mir) SendTelemetry(t proto.Message) error {
	msg, err := tlm_client.GetTelemetryStreamMsg(m.cfg.Device.Id, t)
	if err != nil {
		return err
	}
	return m.sendMsg(msg)
}

// Send proto reported properties to Mir Server
func (m *Mir) SendProperties(t proto.Message) error {
	msg, err := cfg_client.GetReportedPropertiesStreamMsg(m.cfg.Device.Id, t)
	if err != nil {
		return err
	}
	return m.sendMsg(msg)
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
func (m *Mir) HandleCommand(t proto.Message, handler func(proto.Message) (proto.Message, error)) {
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
func (m *Mir) HandleProperties(t proto.Message, handler ...func(proto.Message)) {
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

		var msg proto.Message
		if cfg.t == reflect.TypeFor[dynamicpb.Message]() {
			desc, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(key))
			if err != nil {
				m.l.Error().Err(fmt.Errorf("device error while looking for property descriptor: %w", err))
				return
			}
			msg = dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
		} else {
			v := reflect.New(cfg.t).Interface()
			msg = v.(proto.Message)
		}

		if err := proto.Unmarshal(props.Value, msg); err != nil {
			m.l.Error().Err(err).Msg("error unmarshalling properties")
			return
		}

		for _, handler := range handler {
			go handler(msg)
		}
	}
}

type subject string

// Subject in Mir have the following format:
// device.<device_id>.<module>.<version>.<function>.<extra...>
func (m Mir) NewSubject(module, version, function string, extra ...string) subject {
	extra = append([]string{"device", m.cfg.Device.Id, module, version, function}, extra...)
	return subject(strings.Join(extra, "."))
}

type Header = nats.Header

// Send custom data on a custom route for your own integration.
//
// Use `m.NewSubject` to create a subject.
// Use the module sdk to subscribe to the subject and process the data
func (m *Mir) SendData(sbj subject, data []byte, h Header) error {
	return m.sendMsg(&nats.Msg{
		Subject: string(sbj),
		Header:  h,
		Data:    data,
	})
}

// Send custom json data on a custom route for your own integration.
//
// Use `m.NewSubject` to create a subject.
// Use the module sdk to subscribe to the subject and process the data
func (m *Mir) SendJsonData(sbj subject, data any, h Header) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return m.SendData(sbj, jsonData, h)
}

// Send proto data on a custom route for your own integration.
//
// Use `m.NewSubject` to create a subject.
// Use the module sdk to subscribe to the subject and process the data
func (m *Mir) SendProtoData(sbj subject, data proto.Message, h Header) error {
	protoData, err := proto.Marshal(data)
	if err != nil {
		return err
	}

	if h == nil {
		h = nats.Header{}
	}
	h.Set(HeaderMsgName, string(data.ProtoReflect().Descriptor().FullName()))

	return m.SendData(sbj, protoData, h)
}

// Send proto as json data on a custom route for your own integration.
//
// Use `m.NewSubject` to create a subject.
// Use the module sdk to subscribe to the subject and process the data
func (m *Mir) SendProtoJsonData(sbj subject, data proto.Message, h Header) error {
	jsonData, err := protojson.Marshal(data)
	if err != nil {
		return err
	}

	if h == nil {
		h = nats.Header{}
	}
	h.Set(HeaderMsgName, string(data.ProtoReflect().Descriptor().FullName()))

	return m.SendJsonData(sbj, jsonData, h)
}
