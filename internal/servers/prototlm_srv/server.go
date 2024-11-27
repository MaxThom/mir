package prototlm_srv

import (
	"context"
	"sync"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/services/schema_cache"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoTlmServer struct {
	ctx            context.Context
	tlmStore       ts.TelemetryStore
	sub            *nats.Subscription
	m              *mir.Mir
	devStore       mng.DeviceStore
	devWriters     map[string]map[string]proto_lineprotocol.ProtoBytesToLpFn
	devWritersLock sync.RWMutex
	schStore       *schema_cache.MirProtoCache
}

// TODO prom metics
// - count on number of dev schema
// - count on nb of writers
// - dp count
// - number of device schema fetch

// IDEA clean ingesters map for schema refresh after timespan
// for better schema mng

// Will have to listen to device update event for new
// data that are saved as tag such as namespace and name

type deviceProtoKey struct {
	deviceId     string
	protoMsgName string
}

// DILLEMA could remove all the fields
// and just use the lpPn since each change need to update
// the lpFn

const (
	ServiceName = "mir_telemetry"
)

var (
	uploadMetric = metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	datapointCount = metrics.NewCounter(prometheus.CounterOpts{
		Name: "datapoint_count",
		Help: "Number of datapoint fed into protoproxy from nats",
	})
)

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(uploadMetric)
	reg.Register(datapointCount)
}

func NewProtoTlmServer(logger zerolog.Logger, m *mir.Mir, devStore mng.DeviceStore, tlmStore ts.TelemetryStore) *ProtoTlmServer {
	l = logger.With().Str("srv", "prototlm_server").Logger()
	srv := &ProtoTlmServer{
		tlmStore:   tlmStore,
		m:          m,
		devStore:   devStore,
		devWriters: make(map[string]map[string]proto_lineprotocol.ProtoBytesToLpFn),
		schStore:   schema_cache.NewMirProtoCache(l, m),
	}
	srv.schStore.AddDeviceUpdateSub(srv.handleDeviceUpdate)
	return srv
}

func (s *ProtoTlmServer) handleInfluxErrorChannel() {
	errorsCh := s.tlmStore.Errors()
	go func() {
		for err := range errorsCh {
			l.Error().Err(err).Msg("Error writing to InfluxDB")
		}
	}()
}

func (s *ProtoTlmServer) Listen(ctx context.Context) {
	s.ctx = ctx
	s.handleInfluxErrorChannel()
	s.m.QueueSubscribe(ServiceName, mir.Stream().V1Alpha().Telemetry(s.handleTelemetryStream))
	s.m.QueueSubscribe(ServiceName, mir.Client().V1Alpha().ListTelemetry(s.handleTelemetryListRequest))
}

func (s *ProtoTlmServer) handleTelemetryStream(msg *nats.Msg, deviceId string, protoMsgName string) {
	// TODO prometheus
	// TODO set maximum relidelivery in subscribe
	// TODO handler error with schema if we can't have it.
	// Nak might just create to many relideveries in case of can't find the schema
	// Maybe a buffer zone using channels to connect many routine
	// of this function

	s.devWritersLock.RLock()
	var devWriter proto_lineprotocol.ProtoBytesToLpFn
	devMsgs, ok := s.devWriters[deviceId]
	if ok {
		devWriter, ok = devMsgs[protoMsgName]
	} else {
		s.devWriters[deviceId] = make(map[string]proto_lineprotocol.ProtoBytesToLpFn)
	}
	s.devWritersLock.RUnlock()
	// Mean no ingesters for proto msg, but we might have the schema
	if !ok {
		desc, _, dev, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, protoMsgName, false)
		if err != nil {
			l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
			return
		}
		fn, err := generateIngesters(desc, deviceId, dev)
		// If error, means schema is invalid so request new from device
		if err != nil {
			// TODO possibly different flow depending on error type
			l.Warn().Err(err).Str("deviceId", deviceId).Str("protoMsg", protoMsgName).Msg("Failed to generate ingester function, requesting schema from device")
			desc, _, dev, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, protoMsgName, true)
			if err != nil {
				l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
				return
			}
			fn, err = generateIngesters(desc, deviceId, dev)
			if err != nil {
				l.Warn().Err(err).Msg("")
			}
			// TODO what to do with error here, we cant reask the schema
			// forever if fail, we need maybe a retry of 2-3 times and else
			// it creates an alert in prometheus
		}
		l.Info().Str("deviceId", deviceId).Msg("Generated ingesters functions from proto schema")
		s.devWritersLock.Lock()
		s.devWriters[deviceId][protoMsgName] = fn
		devWriter = fn
		s.devWritersLock.Unlock()
	}
	// TODO update function to return error
	lp := devWriter(msg.Data, map[string]string{})
	// fmt.Println(lp)
	s.tlmStore.WriteDatapoint(lp)
}

func (s *ProtoTlmServer) handleDeviceUpdate(deviceId string, device mir_models.Device, schema mir_proto.MirProtoSchema) {
	l.Debug().Str("device_id", deviceId).Msg("device updated, invalidating device ingesters")
	s.devWritersLock.Lock()
	delete(s.devWriters, deviceId)
	s.devWritersLock.Unlock()
}

type schemaPerDevices struct {
	sch        *mir_proto.MirProtoSchema
	err        error
	devsId     []string
	devsNameNs []string
}

func (s *ProtoTlmServer) handleTelemetryListRequest(msg *nats.Msg, req *tlm_apiv1.SendListTelemetryRequest, e error) {
	l.Info().Any("req", req).Msg("list telemetry request")
	if e != nil {
		l.Error().Err(e).Msg("error occure while receiving request")
		bus.SendProtoReplyOrAck(s.m.Bus, msg, &tlm_apiv1.SendListTelemetryResponse{
			Response: &tlm_apiv1.SendListTelemetryResponse_Error{
				Error: &common_apiv1.Error{
					Code:    400,
					Message: mir_models.ErrorApiDeserializingRequest.Error(),
					Details: []string{"400 Bad Request", e.Error()},
				},
			},
		})
	}

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(e).Msg("error occure while listing devices")
		bus.SendProtoReplyOrAck(s.m.Bus, msg, &tlm_apiv1.SendListTelemetryResponse{
			Response: &tlm_apiv1.SendListTelemetryResponse_Error{
				Error: &common_apiv1.Error{
					Code:    500,
					Message: mir_models.ErrorDbExecutingQuery.Error(),
					Details: []string{"500 Bad Request", e.Error()},
				},
			},
		})
	}

	devsTlm := []*tlm_apiv1.DevicesTelemetry{}
	devSchemas := []*schemaPerDevices{}
	for _, dev := range devs {
		reg, _, err := s.schStore.GetDeviceSchema(dev.Spec.DeviceId, req.RefreshSchema)
		if err != nil {
			found := false
			for _, d := range devsTlm {
				if d.Error == err.Error() {
					d.DevicesNamens = append(d.DevicesNamens, dev.GetNameNamespace())
					found = true
				}
			}
			if !found {
				devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
					DevicesNamens: []string{dev.GetNameNamespace()},
					Error:         err.Error(),
				})
			}
			continue
		}
		found := false
		for _, sch := range devSchemas {
			if mir_proto.AreSchemaEqual(sch.sch, reg) {
				sch.devsId = append(sch.devsId, dev.Spec.DeviceId)
				sch.devsNameNs = append(sch.devsNameNs, dev.GetNameNamespace())
				found = true
			}

		}
		if !found {
			devSchemas = append(devSchemas, &schemaPerDevices{
				sch:        reg,
				devsId:     []string{dev.Spec.DeviceId},
				devsNameNs: []string{dev.GetNameNamespace()},
			})
		}
	}

	for _, sch := range devSchemas {
		tlms, err := sch.sch.GetTelemetryList(req.Measurements, req.Filters)
		if err != nil {
			devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
				DevicesNamens: sch.devsNameNs,
				Error:         err.Error(),
			})
			continue
		}

		for _, tlm := range tlms {
			tlm.Fields, err = s.tlmStore.RetrieveMeasurementsFields(s.ctx, tlm.Name)
			if err != nil {
				tlm.Error = err.Error()
				continue
			}
			tlm.ExploreQuery = s.tlmStore.GetExploreQuery(sch.devsId, tlm.Name)
		}
		devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
			DevicesNamens:  sch.devsNameNs,
			TlmDescriptors: tlms,
		})
	}

	l.Info().Msg("list command request processed successfully")
	bus.SendProtoReplyOrAck(s.m.Bus, msg, &tlm_apiv1.SendListTelemetryResponse{
		Response: &tlm_apiv1.SendListTelemetryResponse_Ok{
			Ok: &tlm_apiv1.TelemetryResponse{
				DevicesTelemetry: devsTlm,
			},
		},
	})
}

// TODO have some device info
func generateIngesters(desc protoreflect.Descriptor, deviceId string, device mir_models.Device) (proto_lineprotocol.ProtoBytesToLpFn, error) {
	tags := map[string]string{
		"__id":        deviceId,
		"__name":      device.Meta.Name,
		"__namespace": device.Meta.Namespace,
	}
	for k, v := range device.Meta.Labels {
		tags["__label_"+k] = v
	}
	fn, err := proto_lineprotocol.GenerateMarshalFn(tags, desc.(protoreflect.MessageDescriptor))
	if err != nil {
		return nil, err
	}

	return fn, err
}
