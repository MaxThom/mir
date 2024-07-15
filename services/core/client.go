package core

import (
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/api/routes"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"google.golang.org/protobuf/proto"
)

const ()

func PublishDeviceCreateRequest(bus *bus.BusConn, req *core.CreateDeviceRequest) (*core.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core.CreateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(routes.CreateDeviceRequest.WithId("todo"), bReq, 7*time.Second)
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

	resMsg, err := bus.Request(routes.UpdateDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
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

	resMsg, err := bus.Request(routes.DeleteDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
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

	resMsg, err := bus.Request(routes.ListDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
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

func PublishHearthbeatStream(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(routes.HearthbeatDeviceStream.WithId(deviceId), []byte{})
}

func PublishDeviceOnlineEvent(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(routes.DeviceOnlineEvent.WithId(deviceId), []byte{})
}

func PublishDeviceOfflineEvent(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(routes.DeviceOfflineEvent.WithId(deviceId), []byte{})
}

func PublishDeviceDeletedEvent(bus *bus.BusConn, deviceId string, d DeviceWithId) error {
	b, err := proto.Marshal(NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(routes.DeviceDeletedEvent.WithId(deviceId), b)
}

func PublishDeviceCreatedEvent(bus *bus.BusConn, deviceId string, d DeviceWithId) error {
	b, err := proto.Marshal(NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(routes.DeviceCreatedEvent.WithId(deviceId), b)
}

func PublishDeviceUpdatedEvent(bus *bus.BusConn, deviceId string, d DeviceWithId) error {
	b, err := proto.Marshal(NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(routes.DeviceUpdatedEvent.WithId(deviceId), b)
}
