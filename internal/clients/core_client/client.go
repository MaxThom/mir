package core_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	CreateDeviceRequest clients.ServerSubject = "client.%s.core.v1alpha.create"
	UpdateDeviceRequest clients.ServerSubject = "client.%s.core.v1alpha.update"
	DeleteDeviceRequest clients.ServerSubject = "client.%s.core.v1alpha.delete"
	ListDeviceRequest   clients.ServerSubject = "client.%s.core.v1alpha.list"

	DeviceOnlineEvent  clients.ServerSubject = "event.%s.core.v1alpha.deviceonline"
	DeviceOfflineEvent clients.ServerSubject = "event.%s.core.v1alpha.deviceoffline"
	DeviceCreatedEvent clients.ServerSubject = "event.%s.core.v1alpha.devicecreated"
	DeviceDeletedEvent clients.ServerSubject = "event.%s.core.v1alpha.devicedeleted"
	DeviceUpdatedEvent clients.ServerSubject = "event.%s.core.v1alpha.deviceupdated"

	HearthbeatDeviceStream clients.ServerSubject = "device.%s.core.v1alpha.hearthbeat"
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

func PublishDeviceOnlineEvent(bus *bus.BusConn, originalInstance string, deviceId string) error {
	msg := &nats.Msg{
		Subject: DeviceOnlineEvent.WithId(deviceId),
		Data:    []byte{},
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}

func PublishDeviceOfflineEvent(bus *bus.BusConn, originalInstance string, deviceId string) error {
	msg := &nats.Msg{
		Subject: DeviceOfflineEvent.WithId(deviceId),
		Data:    []byte{},
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}

func PublishDeviceDeletedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_models.Device) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(d))
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DeviceDeletedEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}

func PublishDeviceCreatedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_models.Device) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(d))
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DeviceCreatedEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}

func PublishDeviceUpdatedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_models.Device) error {
	b, err := proto.Marshal(mir_models.NewProtoDeviceFromDevice(d))
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: DeviceUpdatedEvent.WithId(deviceId),
		Data:    b,
		Header:  nats.Header{},
	}
	msg.Header.Add("original-trigger", originalInstance)
	return bus.PublishMsg(msg)
}
