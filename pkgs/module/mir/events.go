package mir

import (
	"github.com/maxthom/mir/api/routes"
	"github.com/nats-io/nats.go"
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
