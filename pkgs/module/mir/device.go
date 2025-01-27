package mir

import (
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type deviceSubject []string

func (e deviceSubject) String() string {
	return strings.Join(e, ".")
}

type deviceRoutes struct {
	m *Mir
}

// Access all device routes
func (m *Mir) Device() *deviceRoutes {
	return &deviceRoutes{m: m}
}

// Create a Device Route subject to liscen data from a device stream
func (r deviceRoutes) NewSubject(module, version, function string, extra ...string) deviceSubject {
	return append([]string{"device", "*", module, version, function}, extra...)
}

// Listen to a custom stream from devices
// User m.Device().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *deviceRoutes) Subscribe(sbj deviceSubject, h func(msg *Msg, deviceId string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.subscribe(sbj.String(), f)
}

// Listen to a custom stream from devices
// Worker queue behavior means only one worker will process the message
// User m.Device().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *deviceRoutes) QueueSubscribe(queue string, sbj deviceSubject, h func(msg *Msg, deviceId string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.queueSubscribe(queue, sbj.String(), f)
}

/// Hearthbeat

type hearthbeatRoute struct {
	m *Mir
}

// Hearthbeat is used to compute online or offline devices
func (r *deviceRoutes) Hearthbeat() *hearthbeatRoute {
	return &hearthbeatRoute{m: r.m}
}

// Subscribe to hearthbeat messages
// To listen to all devices, use deviceId = "" or deviceId = "*"
// You are responsible of acknowledging the message
func (r *hearthbeatRoute) Subscribe(deviceId string, f func(msg *Msg, deviceId string)) error {
	if deviceId == "" {
		deviceId = "*"
	}
	sbj := core_client.HearthbeatDeviceStream.WithId(deviceId)
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Subscribe to hearthbeat messages
// To listen to all devices, use deviceId = "" or deviceId = "*"
// You are responsible of acknowledging the message
func (r *hearthbeatRoute) QueueSubscribe(queue string, deviceId string, f func(msg *Msg, deviceId string)) error {
	if deviceId == "" {
		deviceId = "*"
	}
	sbj := core_client.HearthbeatDeviceStream.WithId(deviceId)
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *hearthbeatRoute) handlerWrapper(f func(msg *Msg, deviceId string)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
}

/// Telemetry

type telemetryRoute struct {
	m *Mir
}

// Device Telemetry that is stored in influxdb
func (r *deviceRoutes) Telemetry() *telemetryRoute {
	return &telemetryRoute{m: r.m}
}

// Subscribe to telemetry messages
// To listen to all devices, use deviceId = "" or deviceId = "*"
// You are responsible of acknowledging the message
func (r *telemetryRoute) Subscribe(deviceId string, f func(msg *Msg, deviceId string, protoMsgName string, data []byte)) error {
	if deviceId == "" {
		deviceId = "*"
	}
	sbj := tlm_client.TelemetryDeviceStream.WithId(deviceId)
	h := func(msg *nats.Msg) {
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), msg.Header.Get("__msg"), msg.Data)
	}
	return r.m.subscribe(sbj, h)
}

// Subscribe to telemetry messages as a worker queue
// To listen to all devices, use deviceId = "" or deviceId = "*"
// You are responsible of acknowledging the message
func (r *telemetryRoute) QueueSubscribe(queue string, deviceId string, f func(msg *Msg, deviceId string, protoMsgName string, data []byte)) error {
	if deviceId == "" {
		deviceId = "*"
	}
	sbj := tlm_client.TelemetryDeviceStream.WithId(deviceId)
	h := func(msg *nats.Msg) {
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), msg.Header.Get("__msg"), msg.Data)
	}
	return r.m.queueSubscribe(queue, sbj, h)
}

/// Schema

type schemaRoute struct {
	m *Mir
}

// Retrieve device schema
func (r *deviceRoutes) Schema() *schemaRoute {
	return &schemaRoute{m: r.m}
}

// Request device schema
func (r *schemaRoute) Request(deviceId string) (*mir_proto.MirProtoSchema, error) {
	sbj := device_client.SchemaRequest.WithId(deviceId)
	resp, err := r.m.requestWithCompression(sbj, []byte{}, nil, 7*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error requesting device schema: %w", err)
	}

	sch := &device_apiv1.SchemaRetrieveResponse{}
	if err = proto.Unmarshal(resp.Data, sch); err != nil {
		return nil, fmt.Errorf("error deserializing response: %w", err)
	}
	if sch.GetError() != "" {
		return nil, fmt.Errorf("error in device response: %s", sch.GetError())
	}

	return mir_proto.UnmarshalSchema(sch.GetSchema())
}

/// Command

type commandRoute struct {
	m *Mir
}

// Send a command to a device
func (r *deviceRoutes) Command() *commandRoute {
	return &commandRoute{m: r.m}
}

type ProtoCmdDesc struct {
	Name    string
	Payload []byte
}

// Send a command to a device
// ProtoCmdDesc is the message name and the serialized payload
// You need to find the message descriptor fron the schema
// eg:
// desc, err := sch.FindDescriptorByName(protoreflect.FullName(cmdDesc.Name))
// msgResp := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
// err = proto.Unmarshal(cmdDesc.Payload, msgResp)
func (r *commandRoute) Request(deviceId string, cmd proto.Message, timeout time.Duration) (ProtoCmdDesc, error) {
	b, err := proto.Marshal(cmd)
	if err != nil {
		return ProtoCmdDesc{}, fmt.Errorf("error serializing command payload: %w", err)
	}

	return r.RequestRaw(deviceId, ProtoCmdDesc{
		Name:    string(cmd.ProtoReflect().Descriptor().FullName()),
		Payload: b,
	}, timeout)
}

// Send a command to a device with raw payload.
// Useful if your data is already serialized as protobuf
// ProtoCmdDesc is the message name and the serialized payload
// You need to find the message descriptor fron the schema
// eg:
// desc, err := sch.FindDescriptorByName(protoreflect.FullName(cmdDesc.Name))
// msgResp := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
// err = proto.Unmarshal(cmdDesc.Payload, msgResp)
func (r *commandRoute) RequestRaw(deviceId string, cmd ProtoCmdDesc, timeout time.Duration) (ProtoCmdDesc, error) {
	sbj := device_client.CommandRequest.WithId(deviceId)
	h := nats.Header{
		"__msg": []string{cmd.Name},
	}

	resp, err := r.m.request(sbj, cmd.Payload, h, timeout)
	if err != nil {
		return ProtoCmdDesc{}, fmt.Errorf("error requesting device command: %w", err)
	}

	return ProtoCmdDesc{
		Name:    resp.Header.Get("__msg"),
		Payload: resp.Data,
	}, nil
}

/// Config

type configRoute struct {
	m *Mir
}

// Send a config to a device
func (r *deviceRoutes) Config() *configRoute {
	return &configRoute{m: r.m}
}

// Send a config to a device
// ProtoCmdDesc is the message name and the serialized payload
// You need to find the message descriptor fron the schema
// eg:
// desc, err := sch.FindDescriptorByName(protoreflect.FullName(cmdDesc.Name))
// msgResp := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
// err = proto.Unmarshal(cmdDesc.Payload, msgResp)
func (r *configRoute) Publish(deviceId string, cmd proto.Message, t time.Time) error {
	b, err := proto.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("error serializing config payload: %w", err)
	}

	return r.PublishRaw(deviceId, ProtoCmdDesc{
		Name:    string(cmd.ProtoReflect().Descriptor().FullName()),
		Payload: b,
	}, t)
}

// Send a config to a device with raw payload.
// Useful if your data is already serialized as protobuf
// ProtoCmdDesc is the message name and the serialized payload
// You need to find the message descriptor fron the schema
// eg:
// desc, err := sch.FindDescriptorByName(protoreflect.FullName(cmdDesc.Name))
// msgResp := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
// err = proto.Unmarshal(cmdDesc.Payload, msgResp)
func (r *configRoute) PublishRaw(deviceId string, cmd ProtoCmdDesc, t time.Time) error {
	sbj := device_client.ConfigRequest.WithId(deviceId)
	h := nats.Header{
		"__msg":  []string{cmd.Name},
		"__time": []string{t.Format(time.RFC3339Nano)},
	}

	err := r.m.publish(sbj, cmd.Payload, h)
	if err != nil {
		return fmt.Errorf("error requesting device command: %w", err)
	}

	return nil
}
