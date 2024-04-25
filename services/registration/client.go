package registration

import (
	"context"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const (
	createDeviceStream = "client.v1alpha.device.create"
	updateDeviceStream = "client.v1alpha.device.update"
	deleteDeviceStream = "client.v1alpha.device.delete"
	listDeviceStream   = "client.v1alpha.device.list"
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

func PublishDeviceUpdateRequest(ctx context.Context, bus *bus.BusConn, req *registration.UpdateDeviceRequest) (*registration.UpdateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &registration.UpdateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(updateDeviceStream, bReq, 10*time.Second)
	if err != nil {
		return &registration.UpdateDeviceResponse{}, err
	}

	resp := &registration.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &registration.UpdateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceDeleteRequest(ctx context.Context, bus *bus.BusConn, req *registration.DeleteDeviceRequest) (*registration.DeleteDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &registration.DeleteDeviceResponse{}, err
	}

	resMsg, err := bus.Request(updateDeviceStream, bReq, 10*time.Second)
	if err != nil {
		return &registration.DeleteDeviceResponse{}, err
	}

	resp := &registration.DeleteDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &registration.DeleteDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceListRequest(ctx context.Context, bus *bus.BusConn, req *registration.ListDeviceRequest) (*registration.ListDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &registration.ListDeviceResponse{}, err
	}

	resMsg, err := bus.Request(listDeviceStream, bReq, 10*time.Second)
	if err != nil {
		return &registration.ListDeviceResponse{}, err
	}

	resp := &registration.ListDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &registration.ListDeviceResponse{}, err
	}

	return resp, nil
}
