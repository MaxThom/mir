package mir

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/core_client"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout = 7 * time.Second
)

// TODO Eventually, the subject will be a userid when auth is in place
// The serverid is a header

type serverSubject []string

func (e serverSubject) String() string {
	return strings.Join(e, ".")
}

type serverRoutes struct {
	m *Mir
}

// Access all server routes
func (m *Mir) Server() *serverRoutes {
	return &serverRoutes{m: m}
}

// Create a Server Route subject to liscen data from a device stream
func (r serverRoutes) NewSubject(module, version, function string, extra ...string) serverSubject {
	return append([]string{"client", "*", module, version, function}, extra...)
}

// Listen to a custom stream from server
// User m.Server().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *serverRoutes) Subscribe(sbj serverSubject, h func(msg *Msg, clientId string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.subscribe(sbj.String(), f)
}

// Listen to a custom stream from server1
// Worker queue behavior means only one worker will process the message
// User m.Server().NewSubject() to create the subject
// <module>: refer to the module/app your building
// <version>: version of the data in the stream (v1alpha, v1, etc)
// <function>: refer to the exact function of the stream
// <extra>: any extra token you want to add
func (r *serverRoutes) QueueSubscribe(queue string, sbj serverSubject, h func(msg *Msg, clientId string)) error {
	f := func(msg *nats.Msg) {
		h(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId())
	}
	return r.m.queueSubscribe(queue, sbj.String(), f)
}

// Publish proto data to a custom event stream from serve
func (r *serverRoutes) PublishProto(sbj serverSubject, data proto.Message) error {
	sbj[1] = r.m.GetInstanceName()
	b, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{})
}

// Publish json data to a custom event stream from serve
func (r *serverRoutes) PublishJson(sbj serverSubject, data any) error {
	sbj[1] = r.m.GetInstanceName()
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing data: %w", err)

	}
	return r.m.publish(sbj.String(), b, nats.Header{})
}

// Publish data to a custom event stream from serve
func (r *serverRoutes) Publish(sbj serverSubject, data []byte) error {
	sbj[1] = r.m.GetInstanceName()
	return r.m.publish(sbj.String(), data, nats.Header{})
}

/// CreateDevice

type createDeviceRoute struct {
	m *Mir
}

// CreateDevice to integrate a new device in the system
func (r *serverRoutes) CreateDevice() *createDeviceRoute {
	return &createDeviceRoute{m: r.m}
}

// Subscribe to createDevice routes
func (r *createDeviceRoute) Subscribe(f func(msg *Msg, clientId string, d mir_v1.Device) (mir_v1.Device, error)) error {
	sbj := core_client.CreateDeviceRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to createDevice routes
func (r *createDeviceRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, d mir_v1.Device) (mir_v1.Device, error)) error {
	sbj := core_client.CreateDeviceRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *createDeviceRoute) handlerWrapper(f func(msg *Msg, clientId string, d mir_v1.Device) (mir_v1.Device, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.CreateDeviceRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &core_apiv1.CreateDeviceResponse{Response: &core_apiv1.CreateDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_v1.NewDeviceFromCreateDeviceReq(req))
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &core_apiv1.CreateDeviceResponse{Response: &core_apiv1.CreateDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		err = r.m.sendReplyOrAck(msg, &core_apiv1.CreateDeviceResponse{
			Response: &core_apiv1.CreateDeviceResponse_Ok{
				Ok: mir_v1.NewProtoDeviceFromDevice(resp),
			},
		})
	}
}

// Request creation of a new device
func (r *createDeviceRoute) Request(d mir_v1.Device) (mir_v1.Device, error) {
	sbj := core_client.CreateDeviceRequest.WithId(r.m.GetInstanceName())
	req := mir_v1.NewCreateDeviceReqFromDevice(d)
	bReq, err := proto.Marshal(req)
	if err != nil {
		return mir_v1.Device{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return mir_v1.Device{}, err
	}

	resp := &core_apiv1.CreateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return mir_v1.Device{}, err
	}
	if resp.GetError() != "" {
		return mir_v1.Device{}, errors.New(resp.GetError())
	}

	return mir_v1.NewDeviceFromProtoDevice(resp.GetOk()), nil
}

/// UpdateDevice

type updateDeviceRoute struct {
	m *Mir
}

// Update a device in the system
func (r *serverRoutes) UpdateDevice() *updateDeviceRoute {
	return &updateDeviceRoute{m: r.m}
}

// Subscribe to update device routes
func (r *updateDeviceRoute) Subscribe(f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error)) error {
	sbj := core_client.UpdateDeviceRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to update device routes
func (r *updateDeviceRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error)) error {
	sbj := core_client.UpdateDeviceRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *updateDeviceRoute) handlerWrapper(f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.UpdateDeviceRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			_ = r.m.sendReplyOrAck(msg, &core_apiv1.UpdateDeviceResponse{Response: &core_apiv1.UpdateDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_v1.ProtoDeviceTargetToMirDeviceTarget(req.Targets), mir_v1.NewDeviceFromUpdateDeviceReq(req))
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &core_apiv1.UpdateDeviceResponse{Response: &core_apiv1.UpdateDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &core_apiv1.UpdateDeviceResponse{
			Response: &core_apiv1.UpdateDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{Devices: mir_v1.NewProtoDeviceListFromDevices(resp)},
			},
		})
	}
}

// Request update of a device
func (r *updateDeviceRoute) Request(t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error) {
	sbj := core_client.UpdateDeviceRequest.WithId(r.m.GetInstanceName())
	req := mir_v1.DeviceToUpdateDeviceRequest(d)
	req.Targets = mir_v1.MirDeviceTargetToProtoDeviceTarget(t)

	bReq, err := proto.Marshal(req)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resp := &core_apiv1.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Device{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Device{}, errors.New(resp.GetError())
	}

	return mir_v1.NewDeviceListFromProtoDevices(resp.GetOk().Devices), nil
}

// Request update of a device
// Name/Namespace or deviceId must be present to  know which target
func (r *updateDeviceRoute) RequestSingle(d mir_v1.Device) ([]mir_v1.Device, error) {
	sbj := core_client.UpdateDeviceRequest.WithId(r.m.GetInstanceName())
	req := mir_v1.DeviceToUpdateDeviceRequest(d)
	req.Targets = mir_v1.MirDeviceTargetToProtoDeviceTarget(d.ToTarget())

	bReq, err := proto.Marshal(req)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resp := &core_apiv1.UpdateDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Device{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Device{}, errors.New(resp.GetError())
	}

	return mir_v1.NewDeviceListFromProtoDevices(resp.GetOk().Devices), nil
}

/// DeleteDevice

type deleteDeviceRoute struct {
	m *Mir
}

// Delete a device in the system
func (r *serverRoutes) DeleteDevice() *deleteDeviceRoute {
	return &deleteDeviceRoute{m: r.m}
}

// Subscribe to delete device routes
func (r *deleteDeviceRoute) Subscribe(f func(msg *Msg, clientId string, t mir_v1.DeviceTarget) ([]mir_v1.Device, error)) error {
	sbj := core_client.DeleteDeviceRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to delete device routes
func (r *deleteDeviceRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, t mir_v1.DeviceTarget) ([]mir_v1.Device, error)) error {
	sbj := core_client.DeleteDeviceRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deleteDeviceRoute) handlerWrapper(f func(msg *Msg, clientId string, req mir_v1.DeviceTarget) ([]mir_v1.Device, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.DeleteDeviceRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &core_apiv1.DeleteDeviceResponse{Response: &core_apiv1.DeleteDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_v1.ProtoDeviceTargetToMirDeviceTarget(req.Targets))
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &core_apiv1.DeleteDeviceResponse{Response: &core_apiv1.DeleteDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &core_apiv1.DeleteDeviceResponse{
			Response: &core_apiv1.DeleteDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{Devices: mir_v1.NewProtoDeviceListFromDevices(resp)},
			},
		})
	}
}

// Request delete of a device
func (r *deleteDeviceRoute) Request(t mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
	sbj := core_client.DeleteDeviceRequest.WithId(r.m.GetInstanceName())
	req := &core_apiv1.DeleteDeviceRequest{
		Targets: mir_v1.MirDeviceTargetToProtoDeviceTarget(t),
	}
	bReq, err := proto.Marshal(req)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resp := &core_apiv1.DeleteDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Device{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Device{}, errors.New(resp.GetError())
	}

	return mir_v1.NewDeviceListFromProtoDevices(resp.GetOk().Devices), nil
}

/// ListDevice

type listDeviceRoute struct {
	m *Mir
}

// Delete a device in the system
func (r *serverRoutes) ListDevice() *listDeviceRoute {
	return &listDeviceRoute{m: r.m}
}

// Subscribe to list device routes
func (r *listDeviceRoute) Subscribe(f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error)) error {
	sbj := core_client.ListDeviceRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list device routes
func (r *listDeviceRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error)) error {
	sbj := core_client.ListDeviceRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listDeviceRoute) handlerWrapper(f func(msg *Msg, clientId string, t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &core_apiv1.ListDeviceRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &core_apiv1.ListDeviceResponse{Response: &core_apiv1.ListDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_v1.ProtoDeviceTargetToMirDeviceTarget(req.Targets), req.IncludeEvents)
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &core_apiv1.ListDeviceResponse{Response: &core_apiv1.ListDeviceResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &core_apiv1.ListDeviceResponse{
			Response: &core_apiv1.ListDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{Devices: mir_v1.NewProtoDeviceListFromDevices(resp)},
			},
		})
	}
}

// Request list of device
func (r *listDeviceRoute) Request(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
	sbj := core_client.ListDeviceRequest.WithId(r.m.GetInstanceName())
	req := &core_apiv1.ListDeviceRequest{
		Targets:       mir_v1.MirDeviceTargetToProtoDeviceTarget(t),
		IncludeEvents: includeEvents,
	}
	bReq, err := proto.Marshal(req)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Device{}, err
	}

	resp := &core_apiv1.ListDeviceResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Device{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Device{}, errors.New(resp.GetError())
	}

	return mir_v1.NewDeviceListFromProtoDevices(resp.GetOk().Devices), nil
}
