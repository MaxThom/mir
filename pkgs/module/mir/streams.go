package mir

import (
	"github.com/maxthom/mir/api/routes"
	"github.com/nats-io/nats.go"
)

type stream struct{}
type streamV1alpha struct{}

func Stream() stream {
	return stream{}
}

func (s stream) V1Alpha() streamV1alpha {
	return streamV1alpha{}
}

// Hearthbeat stream

type hearthbeatStream struct {
	fn func(msg *nats.Msg, s string)
}

func (s streamV1alpha) Hearthbeat(fn func(msg *nats.Msg, s string)) *hearthbeatStream {
	return &hearthbeatStream{
		fn: fn,
	}
}

func (s hearthbeatStream) subject() string {
	return routes.HearthbeatDeviceStream.WithId("*")
}

func (s hearthbeatStream) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		s.fn(msg, "")
	}
}
