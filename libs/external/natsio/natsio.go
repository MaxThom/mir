package bus

import (
	"time"

	"github.com/nats-io/nats.go"
)

//
// https://github.com/nats-io/nats.go
//

type (
	MirBus = string

	BusConn struct {
		*nats.Conn
		opts []nats.Option
	}
)

var (
	ConfigStreamName    MirBus = "v1alpha_cfg"
	ConfigStreamSubject MirBus = "v1alpha.*.*.cfg.>"
	// <version>.<device_id>.<model_id>.<cfg|tlm|cmd>.<function>
	// v1alpha.XFEA76,FactoryA.cfg.desired_property
	// Could be <name_of_proto_file>.tlm.<version>
	// There could be a service for each proto
	// start with * for now
	TelemetryStreamName    MirBus = "v1alpha_tlm"
	TelemetryStreamSubject MirBus = "v1alpha.*.*.tlm.>"
	TelemetryConsumerProto MirBus = "v1alpha.*.*.tlm.proto"
	// v1alpha.XFEA76.FactoryA.tlm.proto
	// ProtoMessage in header
	CommandStreamName     MirBus = "v1alpha_cmd"
	CommandsStreamSubject MirBus = "v1alpha.*.*.cmd.>"

	// Device
	DeviceStreamName         MirBus = "device"
	DeviceStreamSubject      MirBus = "*.v1alpha.device.*"
	DeviceConsumerCreate     MirBus = "*.v1alpha.device.create"
	DeviceConsumerUpdate     MirBus = "*.v1alpha.device.update"
	DeviceConsunerDelete     MirBus = "*.v1alpha.device.delete"
	DeviceConsumerHearthbeat MirBus = "*.v1alpha.device.hearthbeat"
)

func New(url string, options ...func(*BusConn)) (*BusConn, error) {
	var err error
	bus := &BusConn{}
	for _, o := range options {
		o(bus)
	}

	// TODO Add retry connection here as well
	bus.Conn, err = nats.Connect(url, bus.opts...)

	return bus, err
}

func WithReconnect() func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.MaxReconnects(-1),                                  // Set maximum reconnect attempts
			nats.ReconnectWait(2 * time.Second),                     // Set the wait time between reconnect attempts
			nats.ReconnectJitter(time.Millisecond*100, time.Second), // Set the jitter for reconnects
		}...)
	}
}

func WithReconnHandler(fn nats.ConnHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.ReconnectHandler(fn),
		}...)
	}
}

func WithDisconnHandler(fn nats.ConnErrHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.DisconnectErrHandler(fn),
		}...)
	}
}

func WithClosedHandler(fn nats.ConnHandler) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, []nats.Option{
			nats.ClosedHandler(fn),
		}...)
	}
}

func WithCustom(options ...nats.Option) func(*BusConn) {
	return func(bus *BusConn) {
		bus.opts = append(bus.opts, options...)
	}
}
