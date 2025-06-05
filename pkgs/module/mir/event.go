package mir

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/event_client"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
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

// Access all event routes
func (m *Mir) Event() *eventRoutes {
	return &eventRoutes{m: m}
}

// Create an Event Route subject to liscen data from a device stream.
// Event have the following format:
// event.<id>.<module>.<version>.<function>.<extra...>
//
// <id> is the subject identifier of the event for publishing or specific subscribing.
// Use * for <id> to listen to all events.
func (e eventRoutes) NewSubject(id, module, version, function string, extra ...string) eventSubject {
	return append([]string{"event", id, module, version, function}, extra...)
}

func (e eventRoutes) NewSubjectString(subject string) eventSubject {
	return strings.Split(subject, ".")
}

// Publish an event to the event stream.
// Parameters:
//   - sbj: The event subject containing the routing information for the event
//   - event: The event specification containing the event data to be published
//   - msg: Optional message context for trigger chain propagation. Can be nil if no trigger chain is needed
//
// Returns an error if the event marshaling or publishing fails.
func (r *eventRoutes) Publish(sbj eventSubject, event mir_v1.EventSpec, msg *Msg) error {
	h := nats.Header{}
	if msg != nil && len(msg.GetTriggerChain()) > 0 {
		h[HeaderTrigger] = msg.GetTriggerChain()
	}
	e := mir_v1.MirEventSpecToProtoCreateEvent(event)
	b, err := proto.Marshal(e)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)
	}

	return r.m.publish(sbj.String(), b, h)
}

// Subscribe to all event stream
func (r *eventRoutes) Subscribe(f func(msg *Msg, subjectId string, req mir_v1.EventSpec, e error)) error {
	sbj := event_client.EventsStream.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to event stream
func (r *eventRoutes) QueueSubscribe(queue string, f func(msg *Msg, subjectId string, req mir_v1.EventSpec, e error)) error {
	sbj := event_client.EventsStream.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

// Subscribe to event stream
func (r *eventRoutes) SubscribeSubject(sbj eventSubject, f func(msg *Msg, subjectId string, req mir_v1.EventSpec, e error)) error {
	return r.m.subscribe(sbj.String(), r.handlerWrapper(f))
}

// Queue subscribe to event stream
func (r *eventRoutes) QueueSubscribeSubject(queue string, sbj eventSubject, f func(msg *Msg, subjectId string, req mir_v1.EventSpec, e error)) error {
	return r.m.queueSubscribe(queue, sbj.String(), r.handlerWrapper(f))
}

func (r *eventRoutes) handlerWrapper(f func(msg *Msg, subjectId string, req mir_v1.EventSpec, e error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		subjectId := clients.ServerSubject(msg.Subject).GetId()
		req := &mir_apiv1.CreateEventRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			f(&Msg{msg}, subjectId, mir_v1.EventSpec{}, err)
			return
		}

		f(&Msg{msg}, subjectId, mir_v1.ProtoCreateEventReqToMirEventSpec(req), nil)
	}
}

/// DeviceOnline

type deviceOnlineEventRoute struct {
	m *Mir
}

// Subscribe to device online event routes
func (r *eventRoutes) DeviceOnline() *deviceOnlineEventRoute {
	return &deviceOnlineEventRoute{m: r.m}
}

// Subscribe to device online event routes
func (r *deviceOnlineEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOnlineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceOnlineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOnlineEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		dev := mir_v1.NewDevice()
		if err := eventMsgToObject(msg, &dev); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, nil)
	}
}

/// DeviceOffline

type deviceOfflineEventRoute struct {
	m *Mir
}

// Subscribe to device offline event routes
func (r *eventRoutes) DeviceOffline() *deviceOfflineEventRoute {
	return &deviceOfflineEventRoute{m: r.m}
}

// Subscribe to device online event routes
func (r *deviceOfflineEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device online event routes
func (r *deviceOfflineEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceOfflineEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceOfflineEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		dev := mir_v1.NewDevice()
		if err := eventMsgToObject(msg, &dev); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, nil)
	}
}

/// DeviceCreate

type deviceCreateEventRoute struct {
	m *Mir
}

// Subscribe to device create event routes
func (r *eventRoutes) DeviceCreate() *deviceCreateEventRoute {
	return &deviceCreateEventRoute{m: r.m}
}

// Subscribe to device create event routes
func (r *deviceCreateEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device create event routes
func (r *deviceCreateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceCreatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceCreateEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		dev := mir_v1.NewDevice()
		if err := eventMsgToObject(msg, &dev); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, nil)
	}
}

/// DeviceUpdate

type deviceUpdateEventRoute struct {
	m *Mir
}

// Subscribe to device update event routes
func (r *eventRoutes) DeviceUpdate() *deviceUpdateEventRoute {
	return &deviceUpdateEventRoute{m: r.m}
}

// Subscribe to device update event routes
func (r *deviceUpdateEventRoute) Subscribe(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *deviceUpdateEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceUpdatedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceUpdateEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		dev := mir_v1.NewDevice()
		if err := eventMsgToObject(msg, &dev); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, nil)
	}
}

func getCreateEventRequest(msg *nats.Msg) (*mir_apiv1.CreateEventRequest, error) {
	req := &mir_apiv1.CreateEventRequest{}
	if err := proto.Unmarshal(msg.Data, req); err != nil {
		return nil, err
	}
	return req, nil
}

func eventMsgToObject(msg *nats.Msg, v any) error {
	req, err := getCreateEventRequest(msg)
	if err != nil {
		return err
	}
	if req.Spec != nil {
		err = json.Unmarshal(req.Spec.JsonPayload, v)
		if err != nil {
			return err
		}
	}
	return nil
}

/// DeviceDelete

type deviceDeleteEventRoute struct {
	m *Mir
}

// Subscribe to device delete event routes
func (r *eventRoutes) DeviceDelete() *deviceDeleteEventRoute {
	return &deviceDeleteEventRoute{m: r.m}
}

// Subscribe to device delete event routes
func (r *deviceDeleteEventRoute) Subscribe(f func(msg *Msg, serverId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceDeletedEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device delete event routes
func (r *deviceDeleteEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, device mir_v1.Device, err error)) error {
	sbj := core_client.DeviceDeletedEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deviceDeleteEventRoute) handlerWrapper(f func(msg *Msg, serverId string, deviceId mir_v1.Device, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		dev := mir_v1.NewDevice()
		if err := eventMsgToObject(msg, &dev); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), dev, nil)
	}
}

/// Command Event

type commandEventRoute struct {
	m *Mir
}

// Subscribe to device sent command event
func (r *eventRoutes) Command() *commandEventRoute {
	return &commandEventRoute{m: r.m}
}

// Subscribe to device update event routes
func (r *commandEventRoute) Subscribe(f func(msg *Msg, deviceId string, cmd *mir_apiv1.SendCommandResponse_CommandResponse, err error)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device update event routes
func (r *commandEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, cmd *mir_apiv1.SendCommandResponse_CommandResponse, err error)) error {
	sbj := cmd_client.DeviceCommandEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *commandEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, cmd *mir_apiv1.SendCommandResponse_CommandResponse, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := mir_apiv1.SendCommandResponse_CommandResponse{}
		if err := eventMsgToObject(msg, &req); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), &req, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), &req, nil)
	}
}

// Desired Properties Event

type desiredPropertiesEventRoute struct {
	m *Mir
}

// Subscribe to device desired properties event routes
func (r *eventRoutes) DesiredProperties() *desiredPropertiesEventRoute {
	return &desiredPropertiesEventRoute{m: r.m}
}

// Subscribe to device desired properties event routes
func (r *desiredPropertiesEventRoute) Subscribe(f func(msg *Msg, deviceId string, props map[string]any, err error)) error {
	sbj := cfg_client.DesiredPropertiesEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device desired properties event routes
func (r *desiredPropertiesEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, props map[string]any, err error)) error {
	sbj := cfg_client.DesiredPropertiesEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *desiredPropertiesEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, props map[string]any, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := make(map[string]any)
		// TODO not sure about the correct reference for req here
		if err := eventMsgToObject(msg, &req); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req, nil)
	}
}

// Reported Properties Event

type reportedPropertiesEventRoute struct {
	m *Mir
}

// Subscribe to device reported properties event routes
func (r *eventRoutes) ReportedProperties() *reportedPropertiesEventRoute {
	return &reportedPropertiesEventRoute{m: r.m}
}

// Subscribe to device reported properties event routes
func (r *reportedPropertiesEventRoute) Subscribe(f func(msg *Msg, deviceId string, props map[string]any, err error)) error {
	sbj := cfg_client.ReportedPropertiesEvent.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to device reported properties event routes
func (r *reportedPropertiesEventRoute) QueueSubscribe(queue string, f func(msg *Msg, deviceId string, props map[string]any, err error)) error {
	sbj := cfg_client.ReportedPropertiesEvent.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *reportedPropertiesEventRoute) handlerWrapper(f func(msg *Msg, deviceId string, props map[string]any, err error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := make(map[string]any)
		// TODO not sure about the correct reference for req here
		if err := eventMsgToObject(msg, &req); err != nil {
			f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req, err)
			return
		}
		f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req, nil)
	}
}
