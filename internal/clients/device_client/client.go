package device_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest  clients.DeviceSubject = "%s.v1alpha.schema"
	CommandRequest clients.DeviceSubject = "%s.v1alpha.command"
	ConfigRequest  clients.DeviceSubject = "%s.v1alpha.config"
)

func PublishSchemaRetrieveRequest(bus *nats.Conn, deviceId string) (*mir_apiv1.SchemaRetrieveResponse, error) {
	resMsg, err := bus.Request(SchemaRequest.WithId(deviceId), []byte{}, 10*time.Second)
	if err != nil {
		return &mir_apiv1.SchemaRetrieveResponse{}, err
	}

	data := resMsg.Data
	if resMsg.Header.Get("Content-Encoding") == "zstd" {
		data, err = zstd.DecompressData(resMsg.Data)
		if err != nil {
			return &mir_apiv1.SchemaRetrieveResponse{}, err
		}
	}

	resp := &mir_apiv1.SchemaRetrieveResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return &mir_apiv1.SchemaRetrieveResponse{}, err
	}

	return resp, nil
}
