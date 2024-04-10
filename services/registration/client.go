package registration

import (
	"context"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

var (
	createDeviceStream = "client.v1alpha.device.create"
)

func PublishDeviceCreateRequest(ctx context.Context, bus *bus.BusConn, req *registration.CreateDeviceRequest) (*registration.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &registration.CreateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(createDeviceStream, bReq, 10*time.Second)
	if err != nil {
		return &registration.CreateDeviceResponse{}, err
	}

	resp := &registration.CreateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &registration.CreateDeviceResponse{}, err
	}

	return resp, nil
}
