package mir

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/eventstore_client"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	event_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/event_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type eventSubject []string

func (e eventSubject) String() string {
	return strings.Join(e, ".")
}
func (e eventSubject) WithId(id string) eventSubject {
	e[1] = id
	return e
}

func (e eventSubject) GetId() string {
	return e[1]
}

type eventRoutes struct {
	m *Mir
}

// Access all server routes
func (m *Mir) Event() *eventRoutes {
	return &eventRoutes{m: m}
}

// Create a Event Route subject to liscen data from a device stream
func (r eventRoutes) NewSubject(module, version, function string, extra ...string) eventSubject {
	return append([]string{"event", "*", module, version, function}, extra...)
}

// Publish proto data to a custom event stream from serve
func (r *eventRoutes) PublishProto(sbj eventSubject, originalId string, data proto.Message) error {
	b, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
}

// Publish json data to a custom event stream from serve
func (r *eventRoutes) PublishJson(sbj eventSubject, originalId string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
}

// Publish data to a custom event stream from serve
func (r *eventRoutes) Publish(sbj eventSubject, originalId string, data []byte) error {
	return r.m.publish(sbj.String(), data, nats.Header{HeaderPreviousTrigger: []string{originalId}})
}

func (r *eventRoutes) PublishObject(sbj eventSubject, event mir_models.EventSpec, triggerChain *[]string) error {
	h := nats.Header{}
	if triggerChain != nil && len(*triggerChain) > 0 {
		h[HeaderTrigger] = *triggerChain
	}
	e := mir_models.MirEventSpecToProtoCreateEvent(event)
	b, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)
	}

	return r.m.publish(sbj.String(), b, h)
}

// Subscribe to event stream
func (r *eventRoutes) SubscribeObject(f func(msg *Msg, subjectId string, req mir_models.EventSpec, e error)) error {
	sbj := eventstore_client.EventsStream.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to event stream
func (r *eventRoutes) QueueSubscribeObject(queue string, f func(msg *Msg, subjectId string, req mir_models.EventSpec, e error)) error {
	sbj := eventstore_client.EventsStream.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *eventRoutes) handlerWrapper(f func(msg *Msg, subjectId string, req mir_models.EventSpec, e error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		subjectId := clients.ServerSubject(msg.Subject).GetId()
		req := &event_apiv1.CreateEventRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			f(&Msg{msg}, subjectId, mir_models.EventSpec{}, err)
			return
		}

		f(&Msg{msg}, subjectId, mir_models.ProtoCreateEventReqToMirEventSpec(req), nil)
	}
}

// Listen to a custom event stream from server
// User m.Event().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *eventRoutes) Subscribe(sbj eventSubject, h func(msg *Msg, id string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.subscribe(sbj.String(), f)
}

// Listen to a custom event stream from server
// Worker queue behavior means only one worker will process the message
// User m.Event().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *eventRoutes) QueueSubscribe(queue string, sbj eventSubject, h func(msg *Msg, id string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.queueSubscribe(queue, sbj.String(), f)
}

/// DeviceOnline

type deviceOnlineEventRoute struct {
	m *Mir
}

// Create a new device online event
func (r *eventRoutes) DeviceOnline() *deviceOnlineEventRoute {
	return &deviceOnlineEventRoute{m: r.m}
}

// Subscribe to device online event routes
func (r *deviceOnlineEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOnlineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOnlineEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_models.Device)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.Device{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.NewDeviceFromProtoDevice(req))
	}
}

// Publish a device online event
func (r *deviceOnlineEventRoute) Publish(originalId string, device mir_models.Device) error {
	sbj := core_client.DeviceOnlineEvent.WithId(device.Spec.DeviceId)
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

/// DeviceOffline

type deviceOfflineEventRoute struct {
	m *Mir
}

// Create a new device offline event
func (r *eventRoutes) DeviceOffline() *deviceOfflineEventRoute {
	return &deviceOfflineEventRoute{m: r.m}
}

// Subscribe to device online event routes
func (r *deviceOfflineEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOfflineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOfflineEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_models.Device)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.Device{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.NewDeviceFromProtoDevice(req))
	}
}

// Publish a device online event
func (r *deviceOfflineEventRoute) Publish(originalId string, device mir_models.Device) error {
	sbj := core_client.DeviceOfflineEvent.WithId(device.Spec.DeviceId)
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

/// DeviceCreate

type deviceCreateEventRoute struct {
	m *Mir
}

// Create a new device create event
func (r *eventRoutes) DeviceCreate() *deviceCreateEventRoute {
	return &deviceCreateEventRoute{m: r.m}
}

// Subscribe to device create event routes
func (r *deviceCreateEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device create event routes
func (r *deviceCreateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceCreateEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_models.Device)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.Device{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.NewDeviceFromProtoDevice(req))
	}
}

// Publish a device create event
func (r *deviceCreateEventRoute) Publish(originalId string, device mir_models.Device) error {
	sbj := core_client.DeviceCreatedEvent.WithId(device.Spec.DeviceId)
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

/// DeviceUpdate

type deviceUpdateEventRoute struct {
	m *Mir
}

// Create a new device update event
func (r *eventRoutes) DeviceUpdate() *deviceUpdateEventRoute {
	return &deviceUpdateEventRoute{m: r.m}
}

// Subscribe to device update event routes
func (r *deviceUpdateEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *deviceUpdateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceUpdateEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_models.Device)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.Device{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.NewDeviceFromProtoDevice(req))
	}
}

// Publish a device update event
func (r *deviceUpdateEventRoute) Publish(originalId string, device mir_models.Device) error {
	sbj := core_client.DeviceUpdatedEvent.WithId(device.Spec.DeviceId)
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

/// DeviceDelete

type deviceDeleteEventRoute struct {
	m *Mir
}

// Create a new device delete event
func (r *eventRoutes) DeviceDelete() *deviceDeleteEventRoute {
	return &deviceDeleteEventRoute{m: r.m}
}

// Subscribe to device delete event routes
func (r *deviceDeleteEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceDeletedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device delete event routes
func (r *deviceDeleteEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_models.Device)) error {
	sbj := core_client.DeviceDeletedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceDeleteEventRoute) handlerWrapper(f func(msg *Msg, serverId string, deviceId mir_models.Device)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.Device{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.NewDeviceFromProtoDevice(req))
	}
}

// Publish a device delete event
func (r *deviceDeleteEventRoute) Publish(originalId string, device mir_models.Device) error {
	sbj := core_client.DeviceDeletedEvent.WithId(device.Spec.DeviceId)
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

/// Command Event

type commandEventRoute struct {
	m *Mir
}

// Create a new device update event
func (r *eventRoutes) Command() *commandEventRoute {
	return &commandEventRoute{m: r.m}
}

// Subscribe to device update event routes
func (r *commandEventRoute) Subscribe(f func(msg *Msg, deviceId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *commandEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *commandEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &cmd_apiv1.SendCommandResponse_CommandResponse{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
	}
}

// Publish a device update event
func (r *commandEventRoute) Publish(originalId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse) error {
	sbj := cmd_client.DeviceCommandEvent.WithId(cmd.DeviceId)
	b, err := proto.Marshal(cmd)
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

// Desired Properties Event

type desiredPropertiesEventRoute struct {
	m *Mir
}

// Create a new device desired properties event
func (r *eventRoutes) DesiredProperties() *desiredPropertiesEventRoute {
	return &desiredPropertiesEventRoute{m: r.m}
}

// Subscribe to device desired properties event routes
func (r *desiredPropertiesEventRoute) Subscribe(f func(msg *Msg, deviceId string, props map[string]any)) error {
	sbj := cfg_client.DesiredPropertiesEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device desired properties event routes
func (r *desiredPropertiesEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, props map[string]any)) error {
	sbj := cfg_client.DesiredPropertiesEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *desiredPropertiesEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, props map[string]any)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &structpb.Struct{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req.AsMap())
	}
}

// Publish a device desired properties event
func (r *desiredPropertiesEventRoute) Publish(originalId string, deviceId string, props map[string]any) error {
	s, err := structpb.NewStruct(props)
	if err != nil {
		return err
	}
	b, err := proto.Marshal(s)
	if err != nil {
		return err
	}

	sbj := cfg_client.DesiredPropertiesEvent.WithId(deviceId)
	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}

// Reported Properties Event

type reportedPropertiesEventRoute struct {
	m *Mir
}

// Create a new device reported properties event
func (r *eventRoutes) ReportedProperties() *reportedPropertiesEventRoute {
	return &reportedPropertiesEventRoute{m: r.m}
}

// Subscribe to device reported properties event routes
func (r *reportedPropertiesEventRoute) Subscribe(f func(msg *Msg, deviceId string, props map[string]any)) error {
	sbj := cfg_client.ReportedPropertiesEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device reported properties event routes
func (r *reportedPropertiesEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, props map[string]any)) error {
	sbj := cfg_client.ReportedPropertiesEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *reportedPropertiesEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, props map[string]any)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &structpb.Struct{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req.AsMap())
	}
}

// Publish a device reported properties event
func (r *reportedPropertiesEventRoute) Publish(originalId string, deviceId string, props map[string]any) error {
	s, err := structpb.NewStruct(props)
	if err != nil {
		return err
	}
	b, err := proto.Marshal(s)
	if err != nil {
		return err
	}

	sbj := cfg_client.ReportedPropertiesEvent.WithId(deviceId)
	err = r.m.publish(sbj, b, nats.Header{HeaderPreviousTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}
