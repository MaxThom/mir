package cmd_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	SendCommandRequest  clients.ServerSubject = "client.%s.cmd.v1alpha.send"
	ListCommandsRequest clients.ServerSubject = "client.%s.cmd.v1alpha.list"

	DeviceCommandEvent clients.ServerSubject = "event.%s.core.v1alpha.devicecommand"
)

func PublishSendCommandRequest(bus *bus.BusConn, req *cmd_apiv1.SendCommandRequest) (*cmd_apiv1.SendCommandResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &cmd_apiv1.SendCommandResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(SendCommandRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &cmd_apiv1.SendCommandResponse{}, err
	}

	resp := &cmd_apiv1.SendCommandResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &cmd_apiv1.SendCommandResponse{}, err
	}

	return resp, nil
}

func PublishListCommandsRequest(bus *bus.BusConn, req *cmd_apiv1.SendListCommandsRequest) (*cmd_apiv1.SendListCommandsResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &cmd_apiv1.SendListCommandsResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(ListCommandsRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &cmd_apiv1.SendListCommandsResponse{}, err
	}

	resp := &cmd_apiv1.SendListCommandsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &cmd_apiv1.SendListCommandsResponse{}, err
	}

	return resp, nil
}

func PublishDeviceCommandEvent(bus *nats.Conn, originalInstance string, deviceId string, d *cmd_apiv1.SendCommandResponse_CommandResponse) error {
	b, err := proto.Marshal(d)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DeviceCommandEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("o-instance", originalInstance)
	return bus.PublishMsg(msg)
}
