package device_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/ito/proto/v1alpha/device_ito"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest clients.DeviceSubject = "%s.v1alpha.schema"
)

func PublishSchemaRetreiveRequest(bus *bus.BusConn, deviceId string) (*device_ito.SchemaRetrieveResponse, error) {
	resMsg, err := bus.Request(SchemaRequest.WithId(deviceId), []byte{}, 10*time.Second)
	if err != nil {
		return &device_ito.SchemaRetrieveResponse{}, err
	}

	resp := &device_ito.SchemaRetrieveResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &device_ito.SchemaRetrieveResponse{}, err
	}

	return resp, nil
}
