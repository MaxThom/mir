package swarm

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type swarm struct {
	bus     *bus.BusConn
	Devices []*mirDevice.Mir
}

func NewSwarm(bus *bus.BusConn) swarm {
	return swarm{
		bus: bus,
	}
}

func (s *swarm) Deploy(ctx context.Context) ([]*sync.WaitGroup, error) {
	time.Sleep(2 * time.Second)
	var errs error
	var wgs []*sync.WaitGroup
	for _, d := range s.Devices {
		wg, err := d.Launch(ctx)
		if err != nil {
			errs = errors.Join(err)
		} else {
			wgs = append(wgs, wg)
		}
	}
	return wgs, errs
}

func (s swarm) ToTarget() *core_apiv1.Targets {
	devIds := make([]string, len(s.Devices))
	for i, d := range s.Devices {
		devIds[i] = d.GetDeviceId()
	}
	return &core_apiv1.Targets{
		Ids: devIds,
	}
}

type devicesBuilder struct {
	s          *swarm
	logLevel   mirDevice.LogLevel
	deviceReqs []*core_apiv1.CreateDeviceRequest
	deviceIds  []string
	sch        []protoreflect.FileDescriptor
	cmd        []commandHandler
}

type deviceBuilder struct {
	s         *swarm
	logLevel  mirDevice.LogLevel
	deviceReq *core_apiv1.CreateDeviceRequest
	sch       []protoreflect.FileDescriptor
	cmd       []commandHandler
}

type commandHandler struct {
	target  protoreflect.ProtoMessage
	handler func(protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)
}

func (s *swarm) AddDeviceWithIds(ids []string) *devicesBuilder {
	return &devicesBuilder{
		deviceIds: ids,
		s:         s,
	}
}
func (s *swarm) AddDevices(req ...*core_apiv1.CreateDeviceRequest) *devicesBuilder {
	return &devicesBuilder{
		deviceReqs: req,
		s:          s,
		logLevel:   mirDevice.LogLevelInfo,
	}
}

func (s *swarm) AddDevice(req *core_apiv1.CreateDeviceRequest) *deviceBuilder {
	return &deviceBuilder{
		deviceReq: req,
		s:         s,
		logLevel:  mirDevice.LogLevelInfo,
	}
}

func (b *devicesBuilder) WithSchema(s ...protoreflect.FileDescriptor) *devicesBuilder {
	b.sch = s
	return b
}

func (b *devicesBuilder) WithCommandHandler(t proto.Message, handler func(proto.Message) (proto.Message, error)) *devicesBuilder {
	b.cmd = append(b.cmd, commandHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *devicesBuilder) WithLogLevel(l mirDevice.LogLevel) *devicesBuilder {
	b.logLevel = l
	return b
}

func (b *devicesBuilder) Incubate() ([]*core_apiv1.CreateDeviceResponse, error) {
	var errs error
	for _, d := range b.deviceReqs {
		dev, err := mirDevice.Builder().
			DeviceId(d.Spec.DeviceId).
			LogLevel(b.logLevel).
			Target(b.s.bus.ConnectedUrl()).
			Schema(b.sch...).Build()
		if err != nil {
			errs = errors.Join(err)
			continue
		}
		for _, cmd := range b.cmd {
			dev.HandleCommand(cmd.target, cmd.handler)
		}

		b.s.Devices = append(b.s.Devices, dev)
	}

	for _, d := range b.deviceIds {
		dev, err := mirDevice.Builder().
			DeviceId(d).
			LogLevel(b.logLevel).
			Target(b.s.bus.ConnectedUrl()).
			Schema(b.sch...).Build()
		if err != nil {
			errs = errors.Join(err)
			continue
		}
		for _, cmd := range b.cmd {
			dev.HandleCommand(cmd.target, cmd.handler)
		}

		b.s.Devices = append(b.s.Devices, dev)
	}

	responses := []*core_apiv1.CreateDeviceResponse{}
	for _, reqCreate := range b.deviceReqs {
		resp, err := core_client.PublishDeviceCreateRequest(b.s.bus, reqCreate)
		if err != nil {
			errs = errors.Join(err)
		} else {
			responses = append(responses, resp)
		}
	}
	return responses, errs
}

func (b *deviceBuilder) WithSchema(s ...protoreflect.FileDescriptor) *deviceBuilder {
	b.sch = s
	return b
}

func (b *deviceBuilder) WithCommandHandler(t protoreflect.ProtoMessage, handler func(protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)) *deviceBuilder {
	b.cmd = append(b.cmd, commandHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *deviceBuilder) WithLogLevel(l mirDevice.LogLevel) *deviceBuilder {
	b.logLevel = l
	return b
}

func (b *deviceBuilder) Incubate() (*core_apiv1.CreateDeviceResponse, error) {
	dev, err := mirDevice.Builder().
		DeviceId(b.deviceReq.Spec.DeviceId).
		LogLevel(b.logLevel).
		Target(b.s.bus.ConnectedUrl()).
		Schema(b.sch...).Build()
	if err != nil {
		return nil, err
	}
	for _, cmd := range b.cmd {
		dev.HandleCommand(cmd.target, cmd.handler)
	}
	b.s.Devices = append(b.s.Devices, dev)

	resp, err := core_client.PublishDeviceCreateRequest(b.s.bus, b.deviceReq)
	time.Sleep(2 * time.Second)
	return resp, err
}
