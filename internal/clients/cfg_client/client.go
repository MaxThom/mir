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
	SendConfiguRequest clients.ServerSubject = "client.%s.cfg.v1alpha.send"
	ListConfigRequest  clients.ServerSubject = "client.%s.cfg.v1alpha.list"

	DeviceConfigEvent clients.ServerSubject = "event.%s.cfg.v1alpha.deviceconfig"
)

func PublishSendConfigRequest(bus *bus.BusConn, req *cfg_apiv1.SendConfigRequest) (*cfg_apiv1.SendConfigResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &cfg_apiv1.SendConfigResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(SendConfiguRequest.WithId("todo"), b, 20*time.Second)
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

func PublishDeviceConfigEvent(bus *nats.Conn, originalInstance string, deviceId string, d *cfg_apiv1.SendConfigResponse_ConfigResponse) error {
	b, err := proto.Marshal(d)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DeviceConfigEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}
