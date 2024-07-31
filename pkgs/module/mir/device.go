package mir

import (
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/ito/proto/v1alpha/core_ito"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type request struct{}
type requestV1Alpha struct{}

func Resquest() request {
	return request{}
}

func (r request) V1Alpha() requestV1Alpha {
	return requestV1Alpha{}
}

// Create device request

type createDeviceRequest struct {
	req  *core_ito.CreateDeviceRequest
	resp *core_ito.CreateDeviceResponse
}

func (s requestV1Alpha) CreateDevice(req core_ito.CreateDeviceRequest, resp *core_ito.CreateDeviceResponse) *createDeviceRequest {
	return &createDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *createDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(core_client.CreateDeviceRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *createDeviceRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// Update device request

type updateDeviceRequest struct {
	req  *core_ito.UpdateDeviceRequest
	resp *core_ito.UpdateDeviceResponse
}

func (s requestV1Alpha) UpdateDevice(req core_ito.UpdateDeviceRequest, resp *core_ito.UpdateDeviceResponse) *updateDeviceRequest {
	return &updateDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *updateDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(core_client.UpdateDeviceRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *updateDeviceRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// List device request

type listDeviceRequest struct {
	req  *core_ito.ListDeviceRequest
	resp *core_ito.ListDeviceResponse
}

func (s requestV1Alpha) ListDevice(req core_ito.ListDeviceRequest, resp *core_ito.ListDeviceResponse) *listDeviceRequest {
	return &listDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *listDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(core_client.ListDeviceRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *listDeviceRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// Delete device request

type deleteDeviceRequest struct {
	req  *core_ito.DeleteDeviceRequest
	resp *core_ito.DeleteDeviceResponse
}

func (s requestV1Alpha) DeleteDevice(req core_ito.DeleteDeviceRequest, resp *core_ito.DeleteDeviceResponse) *deleteDeviceRequest {
	return &deleteDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *deleteDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(core_client.DeleteDeviceRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *deleteDeviceRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}
