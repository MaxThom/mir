package cmd_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"google.golang.org/protobuf/proto"
)

const (
	SendCommandRequest  clients.ServerSubject = "client.%s.cmd.v1alpha.send"
	ListCommandsRequest clients.ServerSubject = "client.%s.cmd.v1alpha.list"

	DeviceCommandEvent clients.ServerSubject = "event.%s.cmd.v1alpha.devicecommand"
)

func PublishSendCommandRequest(bus *bus.BusConn, req *mir_apiv1.SendCommandRequest) (*mir_apiv1.SendCommandResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.SendCommandResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(SendCommandRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &mir_apiv1.SendCommandResponse{}, err
	}

	resp := &mir_apiv1.SendCommandResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.SendCommandResponse{}, err
	}

	return resp, nil
}

func PublishListCommandsRequest(bus *bus.BusConn, req *mir_apiv1.SendListCommandsRequest) (*mir_apiv1.SendListCommandsResponse, error) {
	b, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.SendListCommandsResponse{}, err
	}

	// TODO revist timeout
	resMsg, err := bus.Request(ListCommandsRequest.WithId("todo"), b, 20*time.Second)
	if err != nil {
		return &mir_apiv1.SendListCommandsResponse{}, err
	}

	resp := &mir_apiv1.SendListCommandsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.SendListCommandsResponse{}, err
	}

	return resp, nil
}
