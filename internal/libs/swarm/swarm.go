package swarm

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Swarm struct {
	bus     *nats.Conn
	Devices []*mirDevice.Mir
}

func NewSwarm(bus *nats.Conn) Swarm {
	return Swarm{
		bus: bus,
	}
}

func (s *Swarm) BatchDeploy(ctx context.Context, batchSize int) ([]*sync.WaitGroup, error) {
	var errs error
	var wgs []*sync.WaitGroup
	var mu sync.Mutex

	totalDevices := len(s.Devices)
	var batchWg sync.WaitGroup

	// Process devices in batches of 10
	for i := 0; i < totalDevices; i += batchSize {
		end := i + batchSize
		if end > totalDevices {
			end = totalDevices
		}
		if ctx.Err() != nil {
			break
		}

		batch := s.Devices[i:end]
		batchWg.Add(1)
		go func() {
			defer batchWg.Done()
			var wgb []*sync.WaitGroup
			var errsb error
			for _, d := range batch {
				if ctx.Err() != nil {
					return
				}

				wg, err := d.Launch(ctx)
				if err != nil {
					errsb = errors.Join(err)
				} else {
					wgb = append(wgb, wg)
				}
			}

			mu.Lock()
			errs = errors.Join(errsb)
			wgs = append(wgs, wgb...)
			mu.Unlock()
		}()
	}

	batchWg.Wait()
	return wgs, errs
}

func (s *Swarm) Deploy(ctx context.Context) ([]*sync.WaitGroup, error) {
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

func (s Swarm) ToTarget() mir_v1.DeviceTarget {
	devIds := make([]string, len(s.Devices))
	for i, d := range s.Devices {
		devIds[i] = d.GetDeviceId()
	}
	return mir_v1.DeviceTarget{
		Ids: devIds,
	}
}

type devicesBuilder struct {
	s           *Swarm
	logLevel    mirDevice.LogLevel
	credentials string
	tlsKey      string
	tlsCert     string
	caCert      string
	logWriters  []io.Writer
	deviceReqs  []*mir_apiv1.CreateDeviceRequest
	deviceIds   []string
	sch         []protoreflect.FileDescriptor
	schProto    []*descriptorpb.FileDescriptorProto
	cmd         []commandHandler
	cfg         []configHandler
	storeOpts   mirDevice.StoreOptions
}

type deviceBuilder struct {
	s           *Swarm
	credentials string
	tlsKey      string
	tlsCert     string
	caCert      string
	logLevel    mirDevice.LogLevel
	logWriters  []io.Writer
	deviceReq   *mir_apiv1.CreateDeviceRequest
	sch         []protoreflect.FileDescriptor
	schProto    []*descriptorpb.FileDescriptorProto
	cmd         []commandHandler
	cfg         []configHandler
	storeOpts   mirDevice.StoreOptions
}

type commandHandler struct {
	target  protoreflect.ProtoMessage
	handler func(protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error)
}

type configHandler struct {
	target  protoreflect.ProtoMessage
	handler func(protoreflect.ProtoMessage)
}

func (s *Swarm) AddDeviceWithIds(ids []string) *devicesBuilder {
	return &devicesBuilder{
		deviceIds: ids,
		s:         s,
		storeOpts: mirDevice.StoreOptions{
			InMemory: true,
		},
	}
}
func (s *Swarm) AddDevices(req ...*mir_apiv1.CreateDeviceRequest) *devicesBuilder {
	return &devicesBuilder{
		deviceReqs: req,
		s:          s,
		logLevel:   mirDevice.LogLevelInfo,
		storeOpts: mirDevice.StoreOptions{
			InMemory: true,
		},
	}
}

func (s *Swarm) AddDevice(req *mir_apiv1.CreateDeviceRequest) *deviceBuilder {
	return &deviceBuilder{
		deviceReq: req,
		s:         s,
		logLevel:  mirDevice.LogLevelInfo,
		storeOpts: mirDevice.StoreOptions{
			InMemory: true,
		},
	}
}

func (b *devicesBuilder) WithSchema(s ...protoreflect.FileDescriptor) *devicesBuilder {
	b.sch = s
	return b
}

func (b *devicesBuilder) WithSchemaProto(s ...*descriptorpb.FileDescriptorProto) *devicesBuilder {
	b.schProto = s
	return b
}

func (b *devicesBuilder) WithCredentials(filePath string) *devicesBuilder {
	b.credentials = filePath
	return b
}

func (b *devicesBuilder) WithCerticate(keyFile, certFile string) *devicesBuilder {
	b.tlsCert = certFile
	b.tlsKey = keyFile
	return b
}

func (b *devicesBuilder) WithCA(certFile string) *devicesBuilder {
	b.caCert = certFile
	return b
}

func (b *devicesBuilder) WithCommandHandler(t proto.Message, handler func(proto.Message) (proto.Message, error)) *devicesBuilder {
	b.cmd = append(b.cmd, commandHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *devicesBuilder) WithConfigHandler(t proto.Message, handler func(proto.Message)) *devicesBuilder {
	b.cfg = append(b.cfg, configHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *devicesBuilder) WithLogLevel(l mirDevice.LogLevel) *devicesBuilder {
	b.logLevel = l
	return b
}

func (b *devicesBuilder) WithPrettyLogger(colors bool) *devicesBuilder {
	b.logWriters = append(b.logWriters, zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: !colors,
	})
	return b
}

func (b *devicesBuilder) WithStoreOptions(opts mirDevice.StoreOptions) *devicesBuilder {
	b.storeOpts = opts
	return b
}

func (b *devicesBuilder) Incubate() ([]*mir_apiv1.CreateDeviceResponse, error) {
	var errs error
	for _, d := range b.deviceReqs {
		dev, err := mirDevice.Builder().
			DeviceId(d.Spec.DeviceId).
			LogLevel(b.logLevel).
			Store(b.storeOpts).
			Target(b.s.bus.ConnectedUrl()).
			UserCredentialsFile(b.credentials).
			RootCAFile(b.caCert).
			ClientCertificateFile(b.tlsCert, b.tlsKey).
			SchemaProto(b.schProto...).
			Schema(b.sch...).Build()
		if err != nil {
			errs = errors.Join(err)
			continue
		}
		for _, cmd := range b.cmd {
			dev.HandleCommand(cmd.target, cmd.handler)
		}
		for _, cfg := range b.cfg {
			dev.HandleProperties(cfg.target, cfg.handler)
		}

		b.s.Devices = append(b.s.Devices, dev)
	}

	for _, d := range b.deviceIds {
		dev, err := mirDevice.Builder().
			DeviceId(d).
			LogLevel(b.logLevel).
			LogWriters(b.logWriters).
			Store(b.storeOpts).
			Target(b.s.bus.ConnectedUrl()).
			UserCredentialsFile(b.credentials).
			RootCAFile(b.caCert).
			ClientCertificateFile(b.tlsCert, b.tlsKey).
			SchemaProto(b.schProto...).
			Schema(b.sch...).Build()
		if err != nil {
			errs = errors.Join(err)
			continue
		}
		for _, cmd := range b.cmd {
			dev.HandleCommand(cmd.target, cmd.handler)
		}
		for _, cfg := range b.cfg {
			dev.HandleProperties(cfg.target, cfg.handler)
		}

		b.s.Devices = append(b.s.Devices, dev)
	}

	responses := []*mir_apiv1.CreateDeviceResponse{}
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

func (b *deviceBuilder) WithSchemaProto(s ...*descriptorpb.FileDescriptorProto) *deviceBuilder {
	b.schProto = s
	return b
}

func (b *deviceBuilder) WithCredentials(filePath string) *deviceBuilder {
	b.credentials = filePath
	return b
}

func (b *deviceBuilder) WithCerticate(keyFile, certFile string) *deviceBuilder {
	b.tlsCert = certFile
	b.tlsKey = keyFile
	return b
}

func (b *deviceBuilder) WithCA(certFile string) *deviceBuilder {
	b.caCert = certFile
	return b
}

func (b *deviceBuilder) WithCommandHandler(t proto.Message, handler func(proto.Message) (proto.Message, error)) *deviceBuilder {
	b.cmd = append(b.cmd, commandHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *deviceBuilder) WithConfigHandler(t proto.Message, handler func(proto.Message)) *deviceBuilder {
	b.cfg = append(b.cfg, configHandler{
		target:  t,
		handler: handler,
	})
	return b
}

func (b *deviceBuilder) WithLogLevel(l mirDevice.LogLevel) *deviceBuilder {
	b.logLevel = l
	return b
}

func (b *deviceBuilder) WithDevLogger() *deviceBuilder {
	b.logWriters = append(b.logWriters, zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: false,
	})
	return b
}

func (b *deviceBuilder) WithStoreOptions(opts mirDevice.StoreOptions) *deviceBuilder {
	b.storeOpts = opts
	return b
}

func (b *deviceBuilder) Incubate() (*mir_apiv1.CreateDeviceResponse, error) {
	dev, err := mirDevice.Builder().
		DeviceId(b.deviceReq.Spec.DeviceId).
		LogLevel(b.logLevel).
		LogWriters(b.logWriters).
		Store(b.storeOpts).
		Target(b.s.bus.ConnectedUrl()).
		UserCredentialsFile(b.credentials).
		RootCAFile(b.caCert).
		ClientCertificateFile(b.tlsCert, b.tlsKey).
		SchemaProto(b.schProto...).
		Schema(b.sch...).Build()
	if err != nil {
		return nil, err
	}
	for _, cmd := range b.cmd {
		dev.HandleCommand(cmd.target, cmd.handler)
	}
	for _, cfg := range b.cfg {
		dev.HandleProperties(cfg.target, cfg.handler)
	}
	b.s.Devices = append(b.s.Devices, dev)

	resp, err := core_client.PublishDeviceCreateRequest(b.s.bus, b.deviceReq)
	time.Sleep(2 * time.Second)
	return resp, err
}
