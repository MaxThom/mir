package mir

import (
	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/device_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type command struct{}
type commandV1Alpha struct{}

func Command() command {
	return command{}
}

func (r command) V1Alpha() commandV1Alpha {
	return commandV1Alpha{}
}

// Create device request

type retrieveSchemaRequest struct {
	deviceId string
	resp     *device_api.SchemaRetrieveResponse
}

func (s commandV1Alpha) RequestSchema(deviceId string, resp *device_api.SchemaRetrieveResponse) *retrieveSchemaRequest {
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
