package bus

import (
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//
// https://github.com/nats-io/nats.go
//

type (
	MirBus = string

	BusConn struct {
		*nats.Conn
		opts []nats.Option
	}
)

var ()

func New(url string, options ...func(*BusConn)) (*BusConn, error) {
	var err error
	bus := &BusConn{}
	for _, o := range options {
		o(bus)
	}

	// TODO Add retry connection here as well
	bus.Conn, err = nats.Connect(url, bus.opts...)
	// js, _ := bus.Conn.JetStream()
	// kv, _ := js.CreateKeyValue(&nats.KeyValueConfig{
	// 	Bucket: "locks",
	// 	Description: "distributed locking mechanism",
	// 	Storage: nats.MemoryStorage,
	// })

	return bus, err
}

func WithReconnect() func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(-1),                                  // Set maximum reconnect attempts
			nats.ReconnectWait(2 * time.Second),                     // Set the wait time between reconnect attempts
			nats.ReconnectJitter(time.Millisecond*100, time.Second), // Set the jitter for reconnects
		}...)
	}
}

func WithReconnHandler(fn nats.ConnHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.ReconnectHandler(fn),
		}...)
	}
}

func WithDisconnHandler(fn nats.ConnErrHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.DisconnectErrHandler(fn),
		}...)
	}
}

func WithClosedHandler(fn nats.ConnHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.ClosedHandler(fn),
		}...)
	}
}

func WithCustom(options ...nats.Option) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, options...)
	}
}

func SendProtoReplyOrAck(b *nats.Conn, msg *nats.Msg, m protoreflect.ProtoMessage) error {
	if msg.Reply != "" {
		bResp, err := proto.Marshal(m)
		if err != nil {
			return err
		}
		err = b.Publish(msg.Reply, bResp)
		if err != nil {
			return err
		}
	} else {
		msg.Ack()
	}
	return nil
}

func SendReplyOrAck(b *nats.Conn, msg *nats.Msg, data []byte) error {
	if msg.Reply != "" {
		err := b.Publish(msg.Reply, data)
		if err != nil {
			return err
		}
	} else {
		msg.Ack()
	}
	return nil
}
