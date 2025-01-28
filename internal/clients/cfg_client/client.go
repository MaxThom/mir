package cfg_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	SendConfigRequest clients.ServerSubject = "client.%s.cfg.v1alpha.send"
	ListConfigRequest clients.ServerSubject = "client.%s.cfg.v1alpha.list"

	DesiredPropertiesEvent  clients.ServerSubject = "event.%s.cfg.v1alpha.desiredproperties"
	ReportedPropertiesEvent clients.ServerSubject = "event.%s.cfg.v1alpha.reportedproperties"

	ReportedPropertiesStream clients.ServerSubject = "device.%s.cfg.v1alpha.proto"
)

func PublishSendConfigRequest(bus *bus.BusConn, req *cfg_apiv1.SendConfigRequest) (*cfg_apiv1.SendConfigResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &cfg_apiv1.SendConfigResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(SendConfigRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &cfg_apiv1.SendConfigResponse{}, err
	}

	resp := &cfg_apiv1.SendConfigResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &cfg_apiv1.SendConfigResponse{}, err
	}

	return resp, nil
}

func PublishListConfigRequest(bus *bus.BusConn, req *cfg_apiv1.SendListConfigRequest) (*cfg_apiv1.SendListConfigResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &cfg_apiv1.SendListConfigResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(ListConfigRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &cfg_apiv1.SendListConfigResponse{}, err
	}

	resp := &cfg_apiv1.SendListConfigResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &cfg_apiv1.SendListConfigResponse{}, err
	}

	return resp, nil
}

func PublishReportedPropertiesStream(bus *bus.BusConn, deviceId string, t proto.Message) error {
	b, err := proto.Marshal(t)
	if err != nil {
		return err
	}

	return bus.PublishMsg(&nats.Msg{
		Subject: ReportedPropertiesStream.WithId(deviceId),
		Header: nats.Header{
			"__msg": []string{string(t.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	})
}

func PublishDesiredPropertiesEvent(bus *nats.Conn, originalInstance string, deviceId string, d proto.Message) error {
	b, err := proto.Marshal(d)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DesiredPropertiesEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}

func PublishReportedPropertiesEvent(bus *nats.Conn, originalInstance string, deviceId string, d proto.Message) error {
	b, err := proto.Marshal(d)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: ReportedPropertiesEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}
