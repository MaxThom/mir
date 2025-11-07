package core_client

import (
	"fmt"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	CreateDeviceRequest clients.ClientSubject = "client.%s.core.v1alpha.create"
	UpdateDeviceRequest clients.ClientSubject = "client.%s.core.v1alpha.update"
	DeleteDeviceRequest clients.ClientSubject = "client.%s.core.v1alpha.delete"
	ListDeviceRequest   clients.ClientSubject = "client.%s.core.v1alpha.list"

	DeviceOnlineEvent  clients.ClientSubject = "event.%s.core.v1alpha.deviceonline"
	DeviceOfflineEvent clients.ClientSubject = "event.%s.core.v1alpha.deviceoffline"
	DeviceCreatedEvent clients.ClientSubject = "event.%s.core.v1alpha.devicecreated"
	DeviceDeletedEvent clients.ClientSubject = "event.%s.core.v1alpha.devicedeleted"
	DeviceUpdatedEvent clients.ClientSubject = "event.%s.core.v1alpha.deviceupdated"

	HearthbeatDeviceStream clients.ClientSubject = "device.%s.core.v1alpha.hearthbeat"
	SchemaDeviceStream     clients.ClientSubject = "device.%s.core.v1alpha.schema"
)

// Core Builder

func PublishDeviceCreateRequest(bus *nats.Conn, req *mir_apiv1.CreateDeviceRequest) (*mir_apiv1.CreateDeviceResponse, error) {
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

func PublishDeviceUpdateRequest(bus *nats.Conn, req *mir_apiv1.UpdateDeviceRequest) (*mir_apiv1.UpdateDeviceResponse, error) {
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

func PublishDeviceDeleteRequest(bus *nats.Conn, req *mir_apiv1.DeleteDeviceRequest) (*mir_apiv1.DeleteDeviceResponse, error) {
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

func PublishDeviceListRequest(bus *nats.Conn, req *mir_apiv1.ListDeviceRequest) (*mir_apiv1.ListDeviceResponse, error) {
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

func PublishHearthbeatStream(bus *nats.Conn, deviceId string) error {
	return bus.Publish(HearthbeatDeviceStream.WithId(deviceId), []byte{})
}

func PublishHearthbeatWithHello(bus *nats.Conn, deviceId string, sch *descriptorpb.FileDescriptorSet) error {
	schBytes, err := proto.Marshal(sch)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	hello := &mir_apiv1.DeviceHello{
		Response: &mir_apiv1.DeviceHello_Hello{
			Hello: &mir_apiv1.DeviceHelloContent{
				Schema: schBytes,
			},
		},
	}

	bytes, err := proto.Marshal(hello)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	cmprBytes, err := zstd.CompressData(bytes)
	if err != nil {
		return fmt.Errorf("failed to compress schema: %w", err)
	}

	msg := &nats.Msg{
		Subject: HearthbeatDeviceStream.WithId(deviceId),
		Data:    cmprBytes,
		Header: nats.Header{
			"mir-content-encoding": []string{"mir-zstd"},
			"mir-content":          []string{"mir-hello"},
		},
	}
	return bus.PublishMsg(msg)
}

func PublishDeviceDeletedEvent(bus *nats.Conn, originalInstance string, deviceId string, d mir_v1.Device) error {
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

func PublishDeviceCreatedEvent(bus *nats.Conn, originalInstance string, deviceId string, d mir_v1.Device) error {
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

func PublishDeviceUpdatedEvent(bus *nats.Conn, originalInstance string, deviceId string, d mir_v1.Device) error {
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
