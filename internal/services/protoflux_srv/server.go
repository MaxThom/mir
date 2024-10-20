package protoflux_srv

import (
	"context"
	"fmt"
	"sync"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	"github.com/maxthom/mir/internal/mir_utils"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoFluxServer struct {
	tlmStore       ts.TelemetryStore
	sub            *nats.Subscription
	m              *mir.Mir
	devStore       mng.DeviceStore
	devWriters     map[deviceProtoKey]proto_lineprotocol.ProtoBytesToLpFn
	devWritersLock sync.RWMutex
	devSchemas     map[string]deviceProtoSchema
	devSchemasLock sync.RWMutex
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

type deviceProtoSchema struct {
	deviceId   string
	deviceName string
	namespace  string
	labels     map[string]string
	schema     *mir_utils.MirProtoSchema
}

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

func NewProtoFluxServer(logger zerolog.Logger, m *mir.Mir, devStore mng.DeviceStore, tlmStore ts.TelemetryStore) *ProtoFluxServer {
	l = logger.With().Str("srv", "protoflux_server").Logger()
	return &ProtoFluxServer{
		tlmStore:   tlmStore,
		m:          m,
		devStore:   devStore,
		devWriters: make(map[deviceProtoKey]proto_lineprotocol.ProtoBytesToLpFn),
		devSchemas: make(map[string]deviceProtoSchema),
	}
}

func (s *ProtoFluxServer) handleInfluxErrorChannel() {
	errorsCh := s.tlmStore.Errors()
	go func() {
		for err := range errorsCh {
			l.Error().Err(err).Msg("Error writing to InfluxDB")
		}
	}()
}

func (s *ProtoFluxServer) Listen(ctx context.Context) {
	s.handleInfluxErrorChannel()
	s.m.Subscribe(mir.Stream().V1Alpha().Telemetry(
		func(msg *nats.Msg, deviceId string, protoMsgName string) {
			// TODO prometheus
			// TODO set maximum relidelivery in subscribe
			// TODO handler error with schema if we can't have it.
			// Nak might just create to many relideveries in case of can't find the schema
			// Maybe a buffer zone using channels to connect many routine
			// of this function

			s.devWritersLock.RLock()
			devWriter, ok := s.devWriters[deviceProtoKey{
				deviceId:     deviceId,
				protoMsgName: protoMsgName,
			}]
			s.devWritersLock.RUnlock()
			// Mean no ingesters for proto msg, but we might have the schema
			if !ok {
				s.devSchemasLock.RLock()
				devSchema, ok := s.devSchemas[deviceId]
				s.devSchemasLock.RUnlock()
				// No schema, thus we must check in db first
				// if not found, we must request it from device
				if !ok {
					reg, err := mir_utils.ReconcileDeviceSchema(s.m, s.devStore, deviceId, false)
					if err != nil {
						l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
						return
					}
					s.devSchemasLock.Lock()
					devSchema = deviceProtoSchema{
						deviceId: deviceId,
						schema:   reg,
					}
					s.devSchemas[deviceId] = devSchema
					s.devSchemasLock.Unlock()
				}
				fn, err := generateIngesters(devSchema, protoMsgName)
				// If error, means schema is invalid so request new from device
				if err != nil {
					// TODO possibly different flow depending on error type
					l.Warn().Err(err).Str("deviceId", devSchema.deviceId).Str("protoMsg", protoMsgName).Msg("Failed to generate ingester function, requesting schema from device")
					reg, err := mir_utils.ReconcileDeviceSchema(s.m, s.devStore, deviceId, true)
					if err != nil {
						l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
						return
					}
					s.devSchemasLock.Lock()
					devSchema = deviceProtoSchema{
						deviceId: deviceId,
						schema:   reg,
					}
					s.devSchemas[deviceId] = devSchema
					s.devSchemasLock.Unlock()
					fn, err = generateIngesters(devSchema, protoMsgName)
					if err != nil {
						l.Warn().Err(err).Msg("")
					}
					// TODO what to do with error here, we cant reask the schema
					// forever if fail, we need maybe a retry of 2-3 times and else
					// it creates an alert in prometheus
				}
				l.Info().Str("deviceId", deviceId).Msg("Generated ingesters functions from proto schema")
				s.devWritersLock.Lock()
				s.devWriters[deviceProtoKey{
					deviceId:     deviceId,
					protoMsgName: protoMsgName,
				}] = fn
				devWriter = fn
				s.devWritersLock.Unlock()
			}
			// TODO update function to return error
			lp := devWriter(msg.Data, map[string]string{})
			fmt.Println(lp)
			s.tlmStore.WriteDatapoint(lp)
		}))
}

func generateIngesters(devSchema deviceProtoSchema, protoMsgName string) (proto_lineprotocol.ProtoBytesToLpFn, error) {
	// Find the descriptor by name
	desc, err := devSchema.schema.FindDescriptorByName(protoreflect.FullName(protoMsgName))
	if err != nil {
		return nil, err
	}

	tags := map[string]string{
		"deviceId":   devSchema.deviceId,
		"deviceName": devSchema.deviceName,
		"namespace":  devSchema.namespace,
	}
	for k, v := range devSchema.labels {
		tags[k] = v
	}
	fn, err := proto_lineprotocol.GenerateMarshalFn(tags, desc.(protoreflect.MessageDescriptor))
	if err != nil {
		return nil, err
	}

	return fn, err
}
