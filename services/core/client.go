package core

import (
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const (
	createDeviceStream = "client.v1alpha.device.create"
	updateDeviceStream = "client.v1alpha.device.update"
	deleteDeviceStream = "client.v1alpha.device.delete"
	listDeviceStream   = "client.v1alpha.device.list"
)

func PublishDeviceCreateRequest(bus *bus.BusConn, req *core.CreateDeviceRequest) (*core.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core.CreateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(createDeviceStream, bReq, 3*time.Second)
	if err != nil {
		return &core.CreateDeviceResponse{}, err
	}

	resp := &core.CreateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core.CreateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceUpdateRequest(bus *bus.BusConn, req *core.UpdateDeviceRequest) (*core.UpdateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core.UpdateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(updateDeviceStream, bReq, 3*time.Second)
	if err != nil {
		return &core.UpdateDeviceResponse{}, err
	}

	resp := &core.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core.UpdateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceDeleteRequest(bus *bus.BusConn, req *core.DeleteDeviceRequest) (*core.DeleteDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core.DeleteDeviceResponse{}, err
	}

	resMsg, err := bus.Request(updateDeviceStream, bReq, 3*time.Second)
	if err != nil {
		return &core.DeleteDeviceResponse{}, err
	}

	resp := &core.DeleteDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core.DeleteDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceListRequest(bus *bus.BusConn, req *core.ListDeviceRequest) (*core.ListDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core.ListDeviceResponse{}, err
	}

	resMsg, err := bus.Request(listDeviceStream, bReq, 3*time.Second)
	if err != nil {
		return &core.ListDeviceResponse{}, err
	}

	resp := &core.ListDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core.ListDeviceResponse{}, err
	}

	return resp, nil
}
