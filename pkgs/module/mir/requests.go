package mir

import (
	"github.com/maxthom/mir/api/gen/proto/v1alpha/device"
	"github.com/maxthom/mir/api/routes"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type deviceCmd struct{}
type deviceV1Alpha struct{}

func Device() deviceCmd {
	return deviceCmd{}
}

func (r deviceCmd) V1Alpha() deviceV1Alpha {
	return deviceV1Alpha{}
}

// Create device request

type retrieveSchemaRequest struct {
	deviceId string
	resp     *device.SchemaRetrieveResponse
}

func (s deviceV1Alpha) RequestSchema(deviceId string, resp *device.SchemaRetrieveResponse) *retrieveSchemaRequest {
	return &retrieveSchemaRequest{
		deviceId: deviceId,
		resp:     resp,
	}
}

func (s *retrieveSchemaRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(routes.SchemaRequest.WithId(s.deviceId))
	return m, nil
}

func (s *retrieveSchemaRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}
