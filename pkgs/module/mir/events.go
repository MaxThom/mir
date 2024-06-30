package mir

import (
	core_api "github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/api/routes"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type event struct{}
type eventV1alpha struct{}

// Events are server side computed events from
// device telemetry or client request
func Event() event {
	return event{}
}

// All V1Alpha of the data body of the events
func (s event) V1Alpha() eventV1alpha {
	return eventV1alpha{}
}

// Device online event

type deviceOnlineEvent struct {
	fn func(msg *nats.Msg, deviceId string)
}

// Triggered every time a device comes online
// Computed by the core module using the hearthbeat
func (s eventV1alpha) DeviceOnline(fn func(msg *nats.Msg, deviceId string)) *deviceOnlineEvent {
	return &deviceOnlineEvent{
		fn: fn,
	}
}

func (s deviceOnlineEvent) subject() string {
	return routes.DeviceOnlineEvent.WithId("*")
}

func (s deviceOnlineEvent) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		s.fn(msg, routes.Subject(msg.Subject).GetId())
	}
}

// Device offline event

type deviceOfflineEvent struct {
	fn func(msg *nats.Msg, deviceId string)
}

// Triggered every time a device goes offline
// Computed by the core module using the hearthbeat
func (s eventV1alpha) DeviceOffline(fn func(msg *nats.Msg, deviceId string)) *deviceOfflineEvent {
	return &deviceOfflineEvent{
		fn: fn,
	}
}

func (s deviceOfflineEvent) subject() string {
	return routes.DeviceOfflineEvent.WithId("*")
}

func (s deviceOfflineEvent) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		s.fn(msg, routes.Subject(msg.Subject).GetId())
	}
}

// Device deleted event

type deviceDeletedEvent struct {
	fn func(msg *nats.Msg, deviceId string, device *core_api.Device)
}

// Triggered every time a device get deleted
func (s eventV1alpha) DeviceDeleted(fn func(msg *nats.Msg, deviceId string, device *core_api.Device)) *deviceDeletedEvent {
	return &deviceDeletedEvent{
		fn: fn,
	}
}

func (s deviceDeletedEvent) subject() string {
	return routes.DeviceDeletedEvent.WithId("*")
}

func (s deviceDeletedEvent) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		d := &core_api.Device{}
		_ = proto.Unmarshal(msg.Data, d)
		s.fn(msg, routes.Subject(msg.Subject).GetId(), d)
	}
}

// Device created event

type deviceCreatedEvent struct {
	fn func(msg *nats.Msg, deviceId string, device *core_api.Device)
}

// Triggered every time a device get created
func (s eventV1alpha) DeviceCreated(fn func(msg *nats.Msg, deviceId string, device *core_api.Device)) *deviceCreatedEvent {
	return &deviceCreatedEvent{
		fn: fn,
	}
}

func (s deviceCreatedEvent) subject() string {
	return routes.DeviceCreatedEvent.WithId("*")
}

func (s deviceCreatedEvent) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		d := &core_api.Device{}
		_ = proto.Unmarshal(msg.Data, d)
		s.fn(msg, routes.Subject(msg.Subject).GetId(), d)
	}
}

// Device updated event

type deviceUpdatedEvent struct {
	fn func(msg *nats.Msg, deviceId string, device *core_api.Device)
}

// Triggered every time a device get updated
func (s eventV1alpha) DeviceUpdated(fn func(msg *nats.Msg, deviceId string, device *core_api.Device)) *deviceUpdatedEvent {
	return &deviceUpdatedEvent{
		fn: fn,
	}
}

func (s deviceUpdatedEvent) subject() string {
	return routes.DeviceUpdatedEvent.WithId("*")
}

func (s deviceUpdatedEvent) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		d := &core_api.Device{}
		_ = proto.Unmarshal(msg.Data, d)
		s.fn(msg, routes.Subject(msg.Subject).GetId(), d)
	}
}
