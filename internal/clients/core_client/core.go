package core_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"google.golang.org/protobuf/proto"
)

const (
	CreateDeviceRequest clients.Subject = "client.%s.core.v1alpha.create"
	UpdateDeviceRequest clients.Subject = "client.%s.core.v1alpha.update"
	DeleteDeviceRequest clients.Subject = "client.%s.core.v1alpha.delete"
	ListDeviceRequest   clients.Subject = "client.%s.core.v1alpha.list"

	DeviceOnlineEvent  clients.Subject = "event.%s.core.v1alpha.deviceonline"
	DeviceOfflineEvent clients.Subject = "event.%s.core.v1alpha.deviceoffline"
	DeviceCreatedEvent clients.Subject = "event.%s.core.v1alpha.devicecreated"
	DeviceDeletedEvent clients.Subject = "event.%s.core.v1alpha.devicedeleted"
	DeviceUpdatedEvent clients.Subject = "event.%s.core.v1alpha.deviceupdated"

	HearthbeatDeviceStream clients.Subject = "device.%s.core.v1alpha.hearthbeat"
)

// Core Builder

func PublishDeviceCreateRequest(bus *bus.BusConn, req *core_apiv1.CreateDeviceRequest) (*core_apiv1.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core_apiv1.CreateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(CreateDeviceRequest.WithId("todo"), bReq, 7*time.Second)
	if err != nil {
		return &core_apiv1.CreateDeviceResponse{}, err
	}

	resp := &core_apiv1.CreateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core_apiv1.CreateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceUpdateRequest(bus *bus.BusConn, req *core_apiv1.UpdateDeviceRequest) (*core_apiv1.UpdateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core_apiv1.UpdateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(UpdateDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &core_apiv1.UpdateDeviceResponse{}, err
	}

	resp := &core_apiv1.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core_apiv1.UpdateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceDeleteRequest(bus *bus.BusConn, req *core_apiv1.DeleteDeviceRequest) (*core_apiv1.DeleteDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core_apiv1.DeleteDeviceResponse{}, err
	}

	resMsg, err := bus.Request(DeleteDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &core_apiv1.DeleteDeviceResponse{}, err
	}

	resp := &core_apiv1.DeleteDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core_apiv1.DeleteDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceListRequest(bus *bus.BusConn, req *core_apiv1.ListDeviceRequest) (*core_apiv1.ListDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &core_apiv1.ListDeviceResponse{}, err
	}

	resMsg, err := bus.Request(ListDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &core_apiv1.ListDeviceResponse{}, err
	}

	resp := &core_apiv1.ListDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &core_apiv1.ListDeviceResponse{}, err
	}

	return resp, nil
}

func PublishHearthbeatStream(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(HearthbeatDeviceStream.WithId(deviceId), []byte{})
}

func PublishDeviceOnlineEvent(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(DeviceOnlineEvent.WithId(deviceId), []byte{})
}

func PublishDeviceOfflineEvent(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(DeviceOfflineEvent.WithId(deviceId), []byte{})
}

func PublishDeviceDeletedEvent(bus *bus.BusConn, deviceId string, d mir_models.DeviceWithId) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(DeviceDeletedEvent.WithId(deviceId), b)
}

func PublishDeviceCreatedEvent(bus *bus.BusConn, deviceId string, d mir_models.DeviceWithId) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(DeviceCreatedEvent.WithId(deviceId), b)
}

func PublishDeviceUpdatedEvent(bus *bus.BusConn, deviceId string, d mir_models.DeviceWithId) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDeviceWithId(d))
	if err != nil {
		return err
	}
	return bus.Publish(DeviceUpdatedEvent.WithId(deviceId), b)
}
