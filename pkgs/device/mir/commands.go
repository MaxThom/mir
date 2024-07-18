package mir

import (
	"github.com/maxthom/mir/api/gen/proto/v1alpha/device"
	"github.com/maxthom/mir/api/routes"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	cmdHandlers map[string]func(msg *nats.Msg, m *Mir) error
)

func init() {
	cmdHandlers = make(map[string]func(msg *nats.Msg, m *Mir) error)
	cmdHandlers[routes.SchemaRequest.GetVersionAndFunction()] = SchemaRetreiveCmd
}

func SchemaRetreiveCmd(msg *nats.Msg, m *Mir) error {
	bytes, err := proto.Marshal(m.telemetrySchema)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &device.SchemaRetrieveResponse{
			Response: &device.SchemaRetrieveResponse_Error{
				Error: &device.Error{
					Code:    500,
					Message: "error occure while marshalling schema",
					Details: []string{"500 Internal Server Error", err.Error()},
				},
			},
		})
	}

	return sendReplyOrAck(m.b, msg, &device.SchemaRetrieveResponse{
		Response: &device.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	})
}

func sendReplyOrAck(bus *bus.BusConn, msg *nats.Msg, m protoreflect.ProtoMessage) error {
	if msg.Reply != "" {
		bResp, err := proto.Marshal(m)
		if err != nil {
			return err
		}
		err = bus.Publish(msg.Reply, bResp)
		if err != nil {
			return err
		}
	} else {
		return msg.Ack()
	}
	return nil
}
