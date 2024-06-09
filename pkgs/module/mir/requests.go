package mir

import (
	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/api/routes"
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
	req  *core.CreateDeviceRequest
	resp *core.CreateDeviceResponse
}

func (s requestV1Alpha) CreateDevice(req core.CreateDeviceRequest, resp *core.CreateDeviceResponse) *createDeviceRequest {
	return &createDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *createDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(routes.CreateDeviceStream.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *createDeviceRequest) response(m *nats.Msg) error {
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// Update device request

type updateDeviceRequest struct {
	req  *core.UpdateDeviceRequest
	resp *core.UpdateDeviceResponse
}

func (s requestV1Alpha) UpdateDevice(req core.UpdateDeviceRequest, resp *core.UpdateDeviceResponse) *updateDeviceRequest {
	return &updateDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *updateDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(routes.UpdateDeviceStream.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *updateDeviceRequest) response(m *nats.Msg) error {
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// List device request

type listDeviceRequest struct {
	req  *core.ListDeviceRequest
	resp *core.ListDeviceResponse
}

func (s requestV1Alpha) ListDevice(req core.ListDeviceRequest, resp *core.ListDeviceResponse) *listDeviceRequest {
	return &listDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *listDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(routes.ListDeviceStream.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *listDeviceRequest) response(m *nats.Msg) error {
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// Delete device request

type deleteDeviceRequest struct {
	req  *core.DeleteDeviceRequest
	resp *core.DeleteDeviceResponse
}

func (s requestV1Alpha) DeleteDevice(req core.DeleteDeviceRequest, resp *core.DeleteDeviceResponse) *deleteDeviceRequest {
	return &deleteDeviceRequest{
		req:  &req,
		resp: resp,
	}
}

func (s *deleteDeviceRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(routes.DeleteDeviceStream.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *deleteDeviceRequest) response(m *nats.Msg) error {
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}
