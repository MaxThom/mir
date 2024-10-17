package mir

import (
	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type command struct{}
type commandV1Alpha struct{}

func Command() command {
	return command{}
}

func (r command) V1Alpha() commandV1Alpha {
	return commandV1Alpha{}
}

// Retrieve schema from device

type retrieveSchemaRequest struct {
	deviceId string
	resp     *device_apiv1.SchemaRetrieveResponse
}

func (s commandV1Alpha) RequestSchema(deviceId string, resp *device_apiv1.SchemaRetrieveResponse) *retrieveSchemaRequest {
	return &retrieveSchemaRequest{
		deviceId: deviceId,
		resp:     resp,
	}
}

func (s *retrieveSchemaRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(device_client.SchemaRequest.WithId(s.deviceId))
	return m, nil
}

func (s *retrieveSchemaRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	var err error
	data := m.Data
	if m.Header.Get("Content-Encoding") == "zstd" {
		data, err = zstd.DecompressData(m.Data)
		if err != nil {
			return err
		}
	}
	return proto.Unmarshal(data, s.resp)
}

// Send command to device

type sendCommandRequest struct {
	deviceId string
	req      protoreflect.ProtoMessage
	resp     *ProtoCmdDesc
}

type ProtoCmdDesc struct {
	Name    string
	Payload []byte
}

func (s commandV1Alpha) SendCommand(deviceId string, req protoreflect.ProtoMessage, resp *ProtoCmdDesc) *sendCommandRequest {
	return &sendCommandRequest{
		deviceId: deviceId,
		req:      req,
		resp:     resp,
	}
}

func (s *sendCommandRequest) msg() (*nats.Msg, error) {
	b, err := proto.Marshal(s.req)
	if err != nil {
		return nil, err
	}
	// r := &device_apiv1.SendCommandRequest{
	// 	Msg: &device_apiv1.ProtoPayload{
	// 		Name:    string(s.req.ProtoReflect().Descriptor().FullName()),
	// 		Payload: b,
	// 	},
	// }
	// bp, err := proto.Marshal(r)
	// if err != nil {
	// 	return nil, err
	// }

	//m := nats.NewMsg(device_client.CommandRequest.WithId(s.deviceId))
	m := &nats.Msg{
		Subject: device_client.CommandRequest.WithId(s.deviceId),
		Header: nats.Header{
			"__msg": []string{string(s.req.ProtoReflect().Descriptor().FullName())},
		},
		Data: b,
	}
	return m, err
}

func (s *sendCommandRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	var err error
	data := m.Data
	if m.Header.Get("Content-Encoding") == "zstd" {
		data, err = zstd.DecompressData(m.Data)
		if err != nil {
			return err
		}
	}

	s.resp.Name = m.Header.Get("__msg")
	s.resp.Payload = data
	return nil
}
