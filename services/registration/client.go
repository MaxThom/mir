package registration

import (
	"context"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

var (
	createDeviceStream = "client.v1alpha.device.create"
)

func publishDeviceCreateRequest(ctx context.Context, bus *bus.BusConn, req *registration.CreateDeviceRequest) (registration.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return registration.CreateDeviceResponse{}, err
	}
	err = bus.Publish(createDeviceStream, bReq)
	if err != nil {
		return registration.CreateDeviceResponse{}, err
	}
	return registration.CreateDeviceResponse{}, nil
}
