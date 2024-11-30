package mir

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	req  *core_apiv1.CreateDeviceRequest
	resp *core_apiv1.CreateDeviceResponse
}

func (s requestV1Alpha) CreateDevice(req core_apiv1.CreateDeviceRequest, resp *core_apiv1.CreateDeviceResponse) *createDeviceRequest {
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
	req  *core_apiv1.UpdateDeviceRequest
	resp *core_apiv1.UpdateDeviceResponse
}

func (s requestV1Alpha) UpdateDevice(req core_apiv1.UpdateDeviceRequest, resp *core_apiv1.UpdateDeviceResponse) *updateDeviceRequest {
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
	req  *core_apiv1.ListDeviceRequest
	resp *core_apiv1.ListDeviceResponse
}

func (s requestV1Alpha) ListDevice(req core_apiv1.ListDeviceRequest, resp *core_apiv1.ListDeviceResponse) *listDeviceRequest {
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
	req  *core_apiv1.DeleteDeviceRequest
	resp *core_apiv1.DeleteDeviceResponse
}

func (s requestV1Alpha) DeleteDevice(req core_apiv1.DeleteDeviceRequest, resp *core_apiv1.DeleteDeviceResponse) *deleteDeviceRequest {
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

// Send command request
type sendDeviceCommandRequest struct {
	req  *cmd_apiv1.SendCommandRequest
	resp *cmd_apiv1.SendCommandResponse
}

func (s requestV1Alpha) SendDeviceCommand(req *cmd_apiv1.SendCommandRequest, resp *cmd_apiv1.SendCommandResponse) *sendDeviceCommandRequest {
	return &sendDeviceCommandRequest{
		req:  req,
		resp: resp,
	}
}

func (s *sendDeviceCommandRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(cmd_client.SendCommandRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *sendDeviceCommandRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}

	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// Send command request with proto helpers

type sendDeviceCommandRequestProto struct {
	req       *cmd_apiv1.SendCommandRequest
	resp      *cmd_apiv1.SendCommandResponse
	respProto map[string]SendDeviceCommandResponseProto
	t         reflect.Type
}

type SendDeviceCommandRequestProto struct {
	Targets       *core_apiv1.Targets
	Command       proto.Message
	DryRun        bool
	NoValidation  bool
	RefreshSchema bool
	ForcePush     bool
	TimeoutSec    uint32
}

type SendDeviceCommandResponseProto struct {
	DeviceId string
	Error    string
	Status   cmd_apiv1.CommandResponseStatus
	Response proto.Message
}

func (s requestV1Alpha) SendDeviceCommandProto(protoReq *SendDeviceCommandRequestProto, protoResp map[string]SendDeviceCommandResponseProto, responseType proto.Message) *sendDeviceCommandRequestProto {
	// TODO fix error handling
	b, _ := proto.Marshal(protoReq.Command)
	return &sendDeviceCommandRequestProto{
		req: &cmd_apiv1.SendCommandRequest{
			Targets:         protoReq.Targets,
			Name:            string(protoReq.Command.ProtoReflect().Descriptor().FullName()),
			PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
			Payload:         b,
			DryRun:          protoReq.DryRun,
			NoValidation:    protoReq.NoValidation,
			RefreshSchema:   protoReq.RefreshSchema,
			ForcePush:       protoReq.ForcePush,
			TimeoutSec:      protoReq.TimeoutSec,
		},
		resp:      &cmd_apiv1.SendCommandResponse{},
		respProto: protoResp,
		t:         reflect.TypeOf(responseType).Elem(),
	}
}

func (s *sendDeviceCommandRequestProto) msg() (*nats.Msg, error) {
	m := nats.NewMsg(cmd_client.SendCommandRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *sendDeviceCommandRequestProto) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}

	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	if s.resp.GetError() != "" {
		return errors.New(s.resp.GetError())
	}

	for d, dev := range s.resp.GetOk().DeviceResponses {
		v := reflect.New(s.t).Interface()
		respMsg := v.(protoreflect.ProtoMessage)

		if err = proto.Unmarshal(dev.Payload, respMsg); err != nil {
			dev.Error = fmt.Errorf("cannot unmarshal device response payload: %w: %s", err, dev.Error).Error()
		}

		s.respProto[d] = SendDeviceCommandResponseProto{
			DeviceId: dev.DeviceId,
			Error:    dev.Error,
			Status:   dev.Status,
			Response: respMsg,
		}
	}
	return nil
}

// List commands request

type listDeviceCommandsRequest struct {
	req  *cmd_apiv1.SendListCommandsRequest
	resp *cmd_apiv1.SendListCommandsResponse
}

func (s requestV1Alpha) ListDeviceCommands(req *cmd_apiv1.SendListCommandsRequest, resp *cmd_apiv1.SendListCommandsResponse) *listDeviceCommandsRequest {
	return &listDeviceCommandsRequest{
		req:  req,
		resp: resp,
	}
}

func (s *listDeviceCommandsRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(cmd_client.ListCommandsRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *listDeviceCommandsRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}

// List commands request

type listDeviceTelemetryRequest struct {
	req  *tlm_apiv1.SendListTelemetryRequest
	resp *tlm_apiv1.SendListTelemetryResponse
}

func (s requestV1Alpha) ListDeviceTelemetry(req *tlm_apiv1.SendListTelemetryRequest, resp *tlm_apiv1.SendListTelemetryResponse) *listDeviceTelemetryRequest {
	return &listDeviceTelemetryRequest{
		req:  req,
		resp: resp,
	}
}

func (s *listDeviceTelemetryRequest) msg() (*nats.Msg, error) {
	m := nats.NewMsg(tlm_client.TelemetryListRequest.WithId("TODO"))
	bReq, err := proto.Marshal(s.req)

	if err != nil {
		return m, err
	}
	m.Data = bReq

	return m, nil
}

func (s *listDeviceTelemetryRequest) response(m *nats.Msg) error {
	if s.resp == nil {
		return nil
	}
	err := proto.Unmarshal(m.Data, s.resp)
	if err != nil {
		return err
	}
	return nil
}
