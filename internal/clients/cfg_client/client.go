package cfg_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	SendConfigRequest clients.ServerSubject = "client.%s.cfg.v1alpha.send"
	ListConfigRequest clients.ServerSubject = "client.%s.cfg.v1alpha.list"

	DesiredPropertiesEvent  clients.ServerSubject = "event.%s.cfg.v1alpha.desiredproperties"
	ReportedPropertiesEvent clients.ServerSubject = "event.%s.cfg.v1alpha.reportedproperties"

	ReportedPropertiesStream       clients.ServerSubject = "device.%s.cfg.v1alpha.proto"
	RequestDesiredPropertiesStream clients.ServerSubject = "device.%s.cfg.v1alpha.desiredproperties"
)

func PublishSendConfigRequest(bus *nats.Conn, req *mir_apiv1.SendConfigRequest) (*mir_apiv1.SendConfigResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.SendConfigResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(SendConfigRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &mir_apiv1.SendConfigResponse{}, err
	}

	resp := &mir_apiv1.SendConfigResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.SendConfigResponse{}, err
	}

	return resp, nil
}

func PublishListConfigRequest(bus *nats.Conn, req *mir_apiv1.SendListConfigRequest) (*mir_apiv1.SendListConfigResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.SendListConfigResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(ListConfigRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &mir_apiv1.SendListConfigResponse{}, err
	}

	resp := &mir_apiv1.SendListConfigResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.SendListConfigResponse{}, err
	}

	return resp, nil
}

func PublishReportedPropertiesStream(bus *nats.Conn, deviceId string, t proto.Message) error {
	msg, err := GetReportedPropertiesStreamMsg(deviceId, t)
	if err != nil {
		return err
	}

	return bus.PublishMsg(msg)
}

func GetReportedPropertiesStreamMsg(deviceId string, t proto.Message) (*nats.Msg, error) {
	b, err := proto.Marshal(t)
	if err != nil {
		return nil, err
	}

	return &nats.Msg{
		Subject: ReportedPropertiesStream.WithId(deviceId),
		Header: nats.Header{
			"mir-msg": []string{string(t.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	}, nil
}

func PublishRequestDesiredPropertiesStream(bus *nats.Conn, deviceId string) (*mir_apiv1.DeviceReportedPropertiesResponse, error) {
	resMsg, err := bus.Request(RequestDesiredPropertiesStream.WithId(deviceId), []byte{}, 7*time.Second)
	if err != nil {
		return &mir_apiv1.DeviceReportedPropertiesResponse{}, err

	}

	resp := &mir_apiv1.DeviceReportedPropertiesResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.DeviceReportedPropertiesResponse{}, err
	}

	return resp, nil
}
