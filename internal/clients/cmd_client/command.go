package cmd_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	"google.golang.org/protobuf/proto"
)

const (
	SendCommandRequest clients.Subject = "client.%s.cmd.v1alpha.send"
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
