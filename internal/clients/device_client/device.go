package device_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest clients.DeviceSubject = "%s.v1alpha.schema"
)

func PublishSchemaRetreiveRequest(bus *bus.BusConn, deviceId string) (*device_apiv1.SchemaRetrieveResponse, error) {
	resMsg, err := bus.Request(SchemaRequest.WithId(deviceId), []byte{}, 10*time.Second)
	if err != nil {
		return &device_apiv1.SchemaRetrieveResponse{}, err
	}

	data := resMsg.Data
	if resMsg.Header.Get("Content-Encoding") == "zstd" {
		data, err = zstd.DecompressData(resMsg.Data)
		if err != nil {
			return &device_apiv1.SchemaRetrieveResponse{}, err
		}
	}

	resp := &device_apiv1.SchemaRetrieveResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return &device_apiv1.SchemaRetrieveResponse{}, err
	}

	return resp, nil
}
