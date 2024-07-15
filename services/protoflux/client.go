package protoflux

import (
	"github.com/maxthom/mir/api/routes"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func PublishTelemetryStream(bus *bus.BusConn, deviceId string, t protoreflect.ProtoMessage) error {
	b, err := proto.Marshal(t)
	if err != nil {
		return err
	}

	return bus.PublishMsg(&nats.Msg{
		Subject: routes.TelemetryDeviceStream.WithId(deviceId),
		Header: nats.Header{
			"__msg": []string{string(t.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	})
}
