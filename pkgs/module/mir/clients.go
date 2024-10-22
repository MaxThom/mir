package mir

import (
	"github.com/maxthom/mir/internal/clients/cmd_client"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type client struct{}
type clientV1Alpha struct{}

// TODO rename stream to device

// A request coming from a client
func Client() client {
	return client{}
}

// All V1Alpha of the data body of the data
func (s client) V1Alpha() clientV1Alpha {
	return clientV1Alpha{}
}

// List commands request

type listCommandStream struct {
	fn func(msg *nats.Msg, req *cmd_apiv1.SendListCommandsRequest, e error)
}

func (s clientV1Alpha) ListCommands(fn func(msg *nats.Msg, req *cmd_apiv1.SendListCommandsRequest, e error)) *listCommandStream {
	return &listCommandStream{
		fn: fn,
	}
}

func (s listCommandStream) subject() string {
	return cmd_client.ListCommandsRequest.WithId("*")
}

func (s listCommandStream) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		var req cmd_apiv1.SendListCommandsRequest
		s.fn(msg, &req, proto.Unmarshal(msg.Data, &req))
	}
}

// Send command request

type sendCommandStream struct {
	fn func(msg *nats.Msg, req *cmd_apiv1.SendCommandRequest, e error)
}

func (s clientV1Alpha) SendCommand(fn func(msg *nats.Msg, req *cmd_apiv1.SendCommandRequest, e error)) *sendCommandStream {
	return &sendCommandStream{
		fn: fn,
	}
}

func (s sendCommandStream) subject() string {
	return cmd_client.SendCommandRequest.WithId("*")
}

func (s sendCommandStream) handler() nats.MsgHandler {
	return func(msg *nats.Msg) {
		var req cmd_apiv1.SendCommandRequest
		s.fn(msg, &req, proto.Unmarshal(msg.Data, &req))
	}
}
