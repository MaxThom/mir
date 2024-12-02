package mir

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type eventSubject []string

func (e eventSubject) String() string {
	return strings.Join(e, ".")
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
	sbj[1] = r.m.GetInstanceName()
	b, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
}

// Publish json data to a custom event stream from serve
func (r *eventRoutes) PublishJson(sbj eventSubject, originalId string, data any) error {
	sbj[1] = r.m.GetInstanceName()
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
}

// Publish data to a custom event stream from serve
func (r *eventRoutes) Publish(sbj eventSubject, originalId string, data []byte) error {
	sbj[1] = r.m.GetInstanceName()
	return r.m.publish(sbj.String(), data, nats.Header{HeaderOriginalTrigger: []string{originalId}})
}

// Listen to a custom event stream from server
// User m.Event().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *eventRoutes) Subscribe(sbj eventSubject, h func(msg *Msg, serverId string)) error {
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
func (r *eventRoutes) QueueSubscribe(queue string, sbj eventSubject, h func(msg *Msg, serverId string)) error {
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
func (r *deviceOnlineEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOnlineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOnlineEventRoute) handlerWrapper(f func(msg *Msg, serverId string, device mir_models.Device)) nats.MsgHandler {
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
	sbj := core_client.DeviceOnlineEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
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
func (r *deviceOfflineEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOfflineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOfflineEventRoute) handlerWrapper(f func(msg *Msg, serverId string, device mir_models.Device)) nats.MsgHandler {
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
	sbj := core_client.DeviceOfflineEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
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
func (r *deviceCreateEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device create event routes
func (r *deviceCreateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceCreateEventRoute) handlerWrapper(f func(msg *Msg, serverId string, device mir_models.Device)) nats.MsgHandler {
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
	sbj := core_client.DeviceCreatedEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
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
func (r *deviceUpdateEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *deviceUpdateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceUpdateEventRoute) handlerWrapper(f func(msg *Msg, serverId string, device mir_models.Device)) nats.MsgHandler {
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
	sbj := core_client.DeviceUpdatedEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
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
func (r *deviceDeleteEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, device mir_models.Device)) error {
	sbj := core_client.DeviceDeletedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceDeleteEventRoute) handlerWrapper(f func(msg *Msg, serverId string, device mir_models.Device)) nats.MsgHandler {
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
	sbj := core_client.DeviceDeletedEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(device))
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
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
func (r *commandEventRoute) Subscribe(f func(msg *Msg, serverId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *commandEventRoute) QueueSubscribe(queue string, f func(msg *Msg, serverId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *commandEventRoute) handlerWrapper(f func(msg *Msg, serverId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse)) nats.MsgHandler {
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
	sbj := cmd_client.DeviceCommandEvent.WithId(r.m.GetInstanceName())
	b, err := proto.Marshal(cmd)
	if err != nil {
		return err
	}

	err = r.m.publish(sbj, b, nats.Header{HeaderOriginalTrigger: []string{originalId}})
	if err != nil {
		return err
	}

	return nil
}
