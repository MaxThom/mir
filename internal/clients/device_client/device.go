package device_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/device_api"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest clients.DeviceSubject = "%s.v1alpha.schema"
)

func PublishSchemaRetreiveRequest(bus *bus.BusConn, deviceId string) (*device_api.SchemaRetrieveResponse, error) {
	resMsg, err := bus.Request(SchemaRequest.WithId(deviceId), []byte{}, 10*time.Second)
	if err != nil {
		return &device_api.SchemaRetrieveResponse{}, err
	}

	data := resMsg.Data
	if resMsg.Header.Get("Content-Encoding") == "zstd" {
		data, err = zstd.DecompressData(resMsg.Data)
		if err != nil {
			return &device_api.SchemaRetrieveResponse{}, err
		}
	}

	resp := &device_api.SchemaRetrieveResponse{}
	err = proto.Unmarshal(data, resp)
	if err != nil {
		return &device_api.SchemaRetrieveResponse{}, err
	}

	return resp, nil
}
