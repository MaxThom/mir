package core_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
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
	SchemaDeviceStream     clients.ServerSubject = "device.%s.core.v1alpha.schema"
)

// Core Builder

func PublishDeviceCreateRequest(bus *bus.BusConn, req *mir_apiv1.CreateDeviceRequest) (*mir_apiv1.CreateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.CreateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(CreateDeviceRequest.WithId("todo"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.CreateDeviceResponse{}, err
	}

	resp := &mir_apiv1.CreateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.CreateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceUpdateRequest(bus *bus.BusConn, req *mir_apiv1.UpdateDeviceRequest) (*mir_apiv1.UpdateDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.UpdateDeviceResponse{}, err
	}

	resMsg, err := bus.Request(UpdateDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.UpdateDeviceResponse{}, err
	}

	resp := &mir_apiv1.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.UpdateDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceDeleteRequest(bus *bus.BusConn, req *mir_apiv1.DeleteDeviceRequest) (*mir_apiv1.DeleteDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.DeleteDeviceResponse{}, err
	}

	resMsg, err := bus.Request(DeleteDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.DeleteDeviceResponse{}, err
	}

	resp := &mir_apiv1.DeleteDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.DeleteDeviceResponse{}, err
	}

	return resp, nil
}

func PublishDeviceListRequest(bus *bus.BusConn, req *mir_apiv1.ListDeviceRequest) (*mir_apiv1.ListDeviceResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.ListDeviceResponse{}, err
	}
	resMsg, err := bus.Request(ListDeviceRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.ListDeviceResponse{}, err
	}

	resp := &mir_apiv1.ListDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.ListDeviceResponse{}, err
	}

	return resp, nil
}

func PublishHearthbeatStream(bus *bus.BusConn, deviceId string) error {
	return bus.Publish(HearthbeatDeviceStream.WithId(deviceId), []byte{})
}

func PublishDeviceDeletedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_v1.Device) error {
	b, err := proto.Marshal(mir_v1.NewProtoDeviceFromDevice(d))
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

func PublishDeviceCreatedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_v1.Device) error {
	b, err := proto.Marshal(mir_v1.NewProtoDeviceFromDevice(d))
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

func PublishDeviceUpdatedEvent(bus *bus.BusConn, originalInstance string, deviceId string, d mir_v1.Device) error {
	b, err := proto.Marshal(mir_v1.NewProtoDeviceFromDevice(d))
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
