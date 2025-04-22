package eventstore_client

import (
	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/nats-io/nats.go"
)

const (
	ListEventsRequest clients.ServerSubject = "client.%s.events.v1alpha.list"

	EventsStream clients.ServerSubject = "event.*.*.*.*"
)

func PublishEventsStream(bus *bus.BusConn) error {
	msg, err := GetEventsStreamMsg()
	if err != nil {
		return err
	}

	return bus.PublishMsg(msg)
}

func GetEventsStreamMsg() (*nats.Msg, error) {
	return &nats.Msg{
		Subject: EventsStream.WithId("*"),
		Header:  nats.Header{},
		Data:    []byte{},
	}, nil
}
