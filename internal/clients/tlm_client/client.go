package tlm_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	TelemetryDeviceStream clients.ClientSubject = "device.%s.tlm.v1alpha.proto"

	TelemetryListRequest clients.ClientSubject = "client.%s.tlm.v1alpha.list"
)

func PublishTelemetryStream(bus *nats.Conn, deviceId string, t protoreflect.ProtoMessage) error {
	msg, err := GetTelemetryStreamMsg(deviceId, t)
	if err != nil {
		return err
	}

	return bus.PublishMsg(msg)
}

func GetTelemetryStreamMsg(deviceId string, t protoreflect.ProtoMessage) (*nats.Msg, error) {
	b, err := proto.Marshal(t)
	if err != nil {
		return nil, err
	}

	return &nats.Msg{
		Subject: TelemetryDeviceStream.WithId(deviceId),
		Header: nats.Header{
			"mir-msg": []string{string(t.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	}, nil
}

func PublishTelemetryListRequest(bus *nats.Conn, req *mir_apiv1.SendListTelemetryRequest) (*mir_apiv1.SendListTelemetryResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(TelemetryListRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return nil, err
	}

	resp := &mir_apiv1.SendListTelemetryResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return nil, err
	}

	return resp, err
}
