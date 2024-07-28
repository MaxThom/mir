package routes

import (
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/device"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const (
	SchemaRequest DeviceSubject = "%s.v1alpha.schema"
)

// Telemtry Builder
func (s streamClientBuilder) Device() coreClientStream {
	s.stream.module = "device"
	return coreClientStream{
		clientStream: s,
	}
}

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
