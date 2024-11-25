package mir

import (
	"fmt"
	"reflect"

	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	cmdHandlers map[string]func(msg *nats.Msg, m *Mir) error
)

func init() {
	cmdHandlers = make(map[string]func(msg *nats.Msg, m *Mir) error)
	cmdHandlers[device_client.SchemaRequest.GetVersionAndFunction()] = SchemaRetrieveHandler
	cmdHandlers[device_client.CommandRequest.GetVersionAndFunction()] = DefinedCommandHandler
}

func SchemaRetrieveHandler(msg *nats.Msg, m *Mir) error {
	shouldZstd := false
	if msg.Header.Get("request-encoding") == "zstd" {
		shouldZstd = true
	}
	bytes, err := proto.Marshal(m.schema)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &device_apiv1.SchemaRetrieveResponse{
			Response: &device_apiv1.SchemaRetrieveResponse_Error{
				Error: fmt.Sprintf("error occure while marshaiing schema: %s", err.Error()),
			},
		}, nil, false)
	}

	return sendReplyOrAck(m.b, msg, &device_apiv1.SchemaRetrieveResponse{
		Response: &device_apiv1.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, shouldZstd)
}

func DefinedCommandHandler(msg *nats.Msg, m *Mir) error {
	shouldZstd := false
	if msg.Header.Get("request-encoding") == "zstd" {
		shouldZstd = true
	}
	descName := msg.Header.Get("__msg")
	_, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(descName))
	if err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error while looking for command descriptor: %w", err).Error(),
		}, nil, false)
	}

	h, ok := m.cmdHandlers[descName]
	if !ok {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: "device error: no handler for command " + descName + " found",
		}, nil, false)
	}

	v := reflect.New(h.t).Interface()
	cmdMsg := v.(protoreflect.ProtoMessage)

	if err = proto.Unmarshal(msg.Data, cmdMsg); err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error while unmarshalling command payload: %w", err).Error(),
		}, nil, false)
	}

	cmdResp, err := h.h(cmdMsg)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error in command handler: %w", err).Error(),
		}, nil, false)
	}

	if cmdResp == nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Void{}, nil, false)
	}
	return sendReplyOrAck(m.b, msg, cmdResp, nil, shouldZstd)
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
		h.Add("__msg", string(m.ProtoReflect().Descriptor().FullName()))

		data := bResp
		if shouldZstdCompress {
			compressedBytes, err := zstd.CompressData(bResp)
			if err == nil {
				data = compressedBytes
				h.Add("content-encoding", "zstd")
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
