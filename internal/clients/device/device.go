package routes

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/ito/proto/v1alpha/device"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest clients.DeviceSubject = "%s.v1alpha.schema"
)

func PublishSchemaRetreiveRequest(bus *bus.BusConn, deviceId string) (*device.SchemaRetrieveResponse, error) {
	resMsg, err := bus.Request(SchemaRequest.WithId(deviceId), []byte{}, 10*time.Second)
	if err != nil {
		return &device.SchemaRetrieveResponse{}, err
	}

	resp := &device.SchemaRetrieveResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &device.SchemaRetrieveResponse{}, err
	}

	return resp, nil
}
