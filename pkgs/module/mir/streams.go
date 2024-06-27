package mir

import (
	"github.com/maxthom/mir/api/routes"
	"github.com/nats-io/nats.go"
)

type stream struct{}
type streamV1alpha struct{}

// A Stream of data coming from a single or set of devices
func Stream() stream {
	return stream{}
}

// All V1Alpha of the data body of the stream
func (s stream) V1Alpha() streamV1alpha {
	return streamV1alpha{}
}

// Hearthbeat stream

type hearthbeatStream struct {
	fn func(msg *nats.Msg, deviceId string)
}

// Listen to the hearthbeat stream coming from each device
// Used by the system to compute online or offline devices
func (s streamV1alpha) Hearthbeat(fn func(msg *nats.Msg, deviceId string)) *hearthbeatStream {
	return &hearthbeatStream{
		fn: fn,
	}
}

func (s hearthbeatStream) subject() string {
	return routes.HearthbeatDeviceStream.WithId("*")
}

func (s hearthbeatStream) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		s.fn(msg, routes.Subject(msg.Subject).GetId())
	}
}
