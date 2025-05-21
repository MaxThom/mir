package mir

import (
	"encoding/json"
	"fmt"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

/// ListTelemetry

type listTelemetryRoute struct {
	m *Mir
}

// List device telemetry
func (r *serverRoutes) ListTelemetry() *listTelemetryRoute {
	return &listTelemetryRoute{m: r.m}
}

// Subscribe to list telemetry request
func (r *listTelemetryRoute) Subscribe(f func(msg *Msg, clientId string, req *mir_apiv1.SendListTelemetryRequest) ([]*mir_apiv1.DevicesTelemetry, error)) error {
	sbj := tlm_client.TelemetryListRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list telemetry request
func (r *listTelemetryRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *mir_apiv1.SendListTelemetryRequest) ([]*mir_apiv1.DevicesTelemetry, error)) error {
	sbj := tlm_client.TelemetryListRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listTelemetryRoute) handlerWrapper(f func(msg *Msg, clientId string, req *mir_apiv1.SendListTelemetryRequest) ([]*mir_apiv1.DevicesTelemetry, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.SendListTelemetryRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListTelemetryResponse{Response: &mir_apiv1.SendListTelemetryResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListTelemetryResponse{Response: &mir_apiv1.SendListTelemetryResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListTelemetryResponse{
			Response: &mir_apiv1.SendListTelemetryResponse_Ok{
				Ok: &mir_apiv1.TelemetryResponse{
					DevicesTelemetry: resp,
				},
			},
		})
	}
}

// Request listing of telemetry per device
func (r *listTelemetryRoute) Request(req *mir_apiv1.SendListTelemetryRequest) ([]*mir_apiv1.DevicesTelemetry, error) {
	sbj := tlm_client.TelemetryListRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return []*mir_apiv1.DevicesTelemetry{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []*mir_apiv1.DevicesTelemetry{}, err
	}

	resp := &mir_apiv1.SendListTelemetryResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []*mir_apiv1.DevicesTelemetry{}, err
	}
	if resp.GetError() != "" {
		return []*mir_apiv1.DevicesTelemetry{}, err
	}

	return resp.GetOk().DevicesTelemetry, nil
}

/// ListCommand

type listCommandRoute struct {
	m *Mir
}

// List device command
func (r *serverRoutes) ListCommands() *listCommandRoute {
	return &listCommandRoute{m: r.m}
}

// Subscribe to list command request
func (r *listCommandRoute) Subscribe(f func(msg *Msg, clientId string, req *mir_apiv1.SendListCommandsRequest) (map[string]*mir_apiv1.Commands, error)) error {
	sbj := cmd_client.ListCommandsRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list command request
func (r *listCommandRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *mir_apiv1.SendListCommandsRequest) (map[string]*mir_apiv1.Commands, error)) error {
	sbj := cmd_client.ListCommandsRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listCommandRoute) handlerWrapper(f func(msg *Msg, clientId string, req *mir_apiv1.SendListCommandsRequest) (map[string]*mir_apiv1.Commands, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.SendListCommandsRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListCommandsResponse{Response: &mir_apiv1.SendListCommandsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListCommandsResponse{Response: &mir_apiv1.SendListCommandsResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListCommandsResponse{
			Response: &mir_apiv1.SendListCommandsResponse_Ok{
				Ok: &mir_apiv1.DevicesCommands{
					DeviceCommands: resp,
				},
			},
		})
	}
}

// Request listing of command per device
func (r *listCommandRoute) Request(req *mir_apiv1.SendListCommandsRequest) (map[string]*mir_apiv1.Commands, error) {
	sbj := cmd_client.ListCommandsRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}

	resp := &mir_apiv1.SendListCommandsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}
	if resp.GetError() != "" {
		return map[string]*mir_apiv1.Commands{}, err
	}

	return resp.GetOk().DeviceCommands, nil
}

/// SendCommand

type sendCommandRoute struct {
	m *Mir
}

// Send command to device
func (r *serverRoutes) SendCommand() *sendCommandRoute {
	return &sendCommandRoute{m: r.m}
}

// Subscribe to send command request
func (r *sendCommandRoute) Subscribe(f func(msg *Msg, clientId string, req *mir_apiv1.SendCommandRequest) (*mir_apiv1.SendCommandResponse_CommandResponses, error)) error {
	sbj := cmd_client.SendCommandRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to send command request
func (r *sendCommandRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *mir_apiv1.SendCommandRequest) (*mir_apiv1.SendCommandResponse_CommandResponses, error)) error {
	sbj := cmd_client.SendCommandRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *sendCommandRoute) handlerWrapper(f func(msg *Msg, clientId string, req *mir_apiv1.SendCommandRequest) (*mir_apiv1.SendCommandResponse_CommandResponses, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.SendCommandRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendCommandResponse{Response: &mir_apiv1.SendCommandResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendCommandResponse{Response: &mir_apiv1.SendCommandResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendCommandResponse{
			Response: &mir_apiv1.SendCommandResponse_Ok{
				Ok: resp,
			},
		})
	}
}

// Request send a command to device
func (r *sendCommandRoute) Request(req *mir_apiv1.SendCommandRequest) (map[string]*mir_apiv1.SendCommandResponse_CommandResponse, error) {
	sbj := cmd_client.SendCommandRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return map[string]*mir_apiv1.SendCommandResponse_CommandResponse{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return map[string]*mir_apiv1.SendCommandResponse_CommandResponse{}, err
	}

	resp := &mir_apiv1.SendCommandResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return map[string]*mir_apiv1.SendCommandResponse_CommandResponse{}, err
	}
	if resp.GetError() != "" {
		return map[string]*mir_apiv1.SendCommandResponse_CommandResponse{}, err
	}

	return resp.GetOk().DeviceResponses, nil
}

type SendDeviceCommandRequestProto struct {
	Targets       *mir_apiv1.DeviceTarget
	Command       proto.Message
	DryRun        bool
	NoValidation  bool
	RefreshSchema bool
	ForcePush     bool
	TimeoutSec    uint32
}

// Request send a command to device
func (r *sendCommandRoute) RequestProto(req *SendDeviceCommandRequestProto) (map[string]*mir_apiv1.SendCommandResponse_CommandResponse, error) {
	b, _ := proto.Marshal(req.Command)
	return r.Request(&mir_apiv1.SendCommandRequest{
		Targets:         req.Targets,
		Name:            string(req.Command.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         b,
		DryRun:          req.DryRun,
		NoValidation:    req.NoValidation,
		RefreshSchema:   req.RefreshSchema,
		ForcePush:       req.ForcePush,
		TimeoutSec:      req.TimeoutSec,
	})
}

/// ListConfiguration

type listConfigurationRoute struct {
	m *Mir
}

// List device command
func (r *serverRoutes) ListConfig() *listConfigurationRoute {
	return &listConfigurationRoute{m: r.m}
}

// Subscribe to list command request
func (r *listConfigurationRoute) Subscribe(f func(msg *Msg, clientId string, req *mir_apiv1.SendListConfigRequest) (map[string]*mir_apiv1.Configs, error)) error {
	sbj := cfg_client.ListConfigRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list command request
func (r *listConfigurationRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *mir_apiv1.SendListConfigRequest) (map[string]*mir_apiv1.Configs, error)) error {
	sbj := cfg_client.ListConfigRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listConfigurationRoute) handlerWrapper(f func(msg *Msg, clientId string, req *mir_apiv1.SendListConfigRequest) (map[string]*mir_apiv1.Configs, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.SendListConfigRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListConfigResponse{Response: &mir_apiv1.SendListConfigResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListConfigResponse{Response: &mir_apiv1.SendListConfigResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendListConfigResponse{
			Response: &mir_apiv1.SendListConfigResponse_Ok{
				Ok: &mir_apiv1.DevicesConfigs{
					DeviceConfigs: resp,
				},
			},
		})
	}
}

// Request listing of command per device
func (r *listConfigurationRoute) Request(req *mir_apiv1.SendListCommandsRequest) (map[string]*mir_apiv1.Commands, error) {
	sbj := cfg_client.ListConfigRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}

	resp := &mir_apiv1.SendListCommandsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return map[string]*mir_apiv1.Commands{}, err
	}
	if resp.GetError() != "" {
		return map[string]*mir_apiv1.Commands{}, err
	}

	return resp.GetOk().DeviceCommands, nil
}

/// SendConfig

type sendConfigRoute struct {
	m *Mir
}

// Send config to device
func (r *serverRoutes) SendConfig() *sendConfigRoute {
	return &sendConfigRoute{m: r.m}
}

// Subscribe to send command request
func (r *sendConfigRoute) Subscribe(f func(msg *Msg, clientId string, req *mir_apiv1.SendConfigRequest) (*mir_apiv1.SendConfigResponse_ConfigResponses, error)) error {
	sbj := cfg_client.SendConfigRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to send command request
func (r *sendConfigRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *mir_apiv1.SendConfigRequest) (*mir_apiv1.SendConfigResponse_ConfigResponses, error)) error {
	sbj := cfg_client.SendConfigRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *sendConfigRoute) handlerWrapper(f func(msg *Msg, clientId string, req *mir_apiv1.SendConfigRequest) (*mir_apiv1.SendConfigResponse_ConfigResponses, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.SendConfigRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendConfigResponse{Response: &mir_apiv1.SendConfigResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.SendConfigResponse{Response: &mir_apiv1.SendConfigResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &mir_apiv1.SendConfigResponse{
			Response: &mir_apiv1.SendConfigResponse_Ok{
				Ok: resp,
			},
		})
	}
}

// Request send a command to device
func (r *sendConfigRoute) Request(req *mir_apiv1.SendConfigRequest) (map[string]*mir_apiv1.SendConfigResponse_ConfigResponse, error) {
	sbj := cfg_client.SendConfigRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, err
	}

	resp := &mir_apiv1.SendConfigResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, err
	}
	if resp.GetError() != "" {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, err
	}

	return resp.GetOk().DeviceResponses, nil
}

type SendDeviceConfigRequestProto struct {
	Targets           *mir_apiv1.DeviceTarget
	Command           proto.Message
	DryRun            bool
	RefreshSchema     bool
	ForcePush         bool
	SendOnlyDifferent bool
}

// Request send a config to device using proto data
func (r *sendConfigRoute) RequestProto(req *SendDeviceConfigRequestProto) (map[string]*mir_apiv1.SendConfigResponse_ConfigResponse, error) {
	b, err := proto.Marshal(req.Command)
	if err != nil {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, fmt.Errorf("error marshalling command to proto: %w", err)
	}
	return r.Request(&mir_apiv1.SendConfigRequest{
		Targets:           req.Targets,
		Name:              string(req.Command.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:           b,
		DryRun:            req.DryRun,
		RefreshSchema:     req.RefreshSchema,
		ForcePush:         req.ForcePush,
		SendOnlyDifferent: req.SendOnlyDifferent,
	})
}

type SendDeviceConfigRequestJson struct {
	Targets           *mir_apiv1.DeviceTarget
	CommandName       string
	CommandPayload    any
	DryRun            bool
	RefreshSchema     bool
	ForcePush         bool
	SendOnlyDifferent bool
}

// Request send a config to device using json data
func (r *sendConfigRoute) RequestJson(req *SendDeviceConfigRequestJson) (map[string]*mir_apiv1.SendConfigResponse_ConfigResponse, error) {
	b, err := json.Marshal(req.CommandPayload)
	if err != nil {
		return map[string]*mir_apiv1.SendConfigResponse_ConfigResponse{}, fmt.Errorf("error marshalling command to json: %w", err)
	}
	return r.Request(&mir_apiv1.SendConfigRequest{
		Targets:           req.Targets,
		Name:              req.CommandName,
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           b,
		DryRun:            req.DryRun,
		RefreshSchema:     req.RefreshSchema,
		ForcePush:         req.ForcePush,
		SendOnlyDifferent: req.SendOnlyDifferent,
	})
}
