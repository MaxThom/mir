package routes

import (
	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	TelemetryDeviceStream clients.Subject = "device.%s.telemetry.v1alpha.proto"
)

func PublishTelemetryStream(bus *bus.BusConn, deviceId string, t protoreflect.ProtoMessage) error {
	b, err := proto.Marshal(t)
	if err != nil {
		return err
	}

	return bus.PublishMsg(&nats.Msg{
		Subject: TelemetryDeviceStream.WithId(deviceId),
		Header: nats.Header{
			"__msg": []string{string(t.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	})
}
