package mir

import (
	"github.com/maxthom/mir/api/gen/proto/v1alpha/device"
	"github.com/maxthom/mir/api/routes"
	"github.com/maxthom/mir/libs/compression/zstd"
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
		}, nil, false)
	}

	return sendReplyOrAck(m.b, msg, &device.SchemaRetrieveResponse{
		Response: &device.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, true)
}

func sendReplyOrAck(bus *bus.BusConn, msg *nats.Msg, m protoreflect.ProtoMessage, h nats.Header, shouldZstdCompress bool) error {
	if msg.Reply != "" {
		if h == nil {
			h = nats.Header{}
		}
		bResp, err := proto.Marshal(m)
		if err != nil {
			return err
		}

		data := bResp
		if shouldZstdCompress {
			compressedBytes, err := zstd.CompressData(bResp)
			if err == nil {
				data = compressedBytes
				h.Add("Content-Encoding", "zstd")
			}
		}

		err = bus.PublishMsg(&nats.Msg{
			Subject: msg.Reply,
			Header:  h,
			Data:    data,
		})
		if err != nil {
			return err
		}
	} else {
		return msg.Ack()
	}
	return nil
}
