package protoflux_srv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/device_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
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
	schema     *protoregistry.Files
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
					reg, err := s.reconcileDeviceSchema(deviceId, false)
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
					reg, err := s.reconcileDeviceSchema(deviceId, true)
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
					fn, _ = generateIngesters(devSchema, protoMsgName)
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

func (s *ProtoFluxServer) reconcileDeviceSchema(deviceId string, forceDeviceFetch bool) (*protoregistry.Files, error) {
	// 1. Go get schema in surrealdb
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	if !forceDeviceFetch {
		devs, err := s.devStore.ListDevice(&core_api.ListDeviceRequest{
			Targets: &core_api.Targets{
				Ids: []string{deviceId},
			},
		})
		if err != nil {
			return nil, err
		}
		if len(devs) > 0 {
			if devs[0].Status.Schema.CompressedSchema != nil &&
				len(devs[0].Status.Schema.CompressedSchema) != 0 {
				reg, err := mir_models.DecompressFileDescriptorSet(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					l.Debug().Str("deviceId", deviceId).Msg("Found proto schema from device in database")
					return reg, nil
				} else {
					l.Error().Err(err).Msg("Error retrieving schema from database")
				}
			}
		}
	}

	l.Info().Str("deviceId", deviceId).Msg("Requesting proto schema from device")
	reg, pbSet, err := getProtoSchemaFromDevice(s.m, deviceId)
	if err != nil {
		return nil, err
	}

	// Mainly for extra info
	packNames := []string{}
	reg.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		packNames = append(packNames, string(f.FullName()))
		return true
	})

	compSch, err := mir_models.CompressFileDescriptorSet(pbSet)
	if err != nil {
		return nil, err
	}

	_, err = s.devStore.UpdateDevice(&core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{deviceId},
		},
		Status: &core_api.UpdateDeviceRequest_Status{
			Schema: &core_api.UpdateDeviceRequest_Schema{
				CompressedSchema: compSch,
				PackageNames:     packNames,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(time.Now().UTC()),
			},
		},
	})

	return reg, err
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

func getProtoSchemaFromDevice(m *mir.Mir, deviceId string) (*protoregistry.Files, *descriptorpb.FileDescriptorSet, error) {
	schemaResp := &device_api.SchemaRetrieveResponse{}
	err := m.SendRequest(mir.Command().V1Alpha().RequestSchema(deviceId, schemaResp))
	if err != nil {
		return nil, nil, err
	} else if schemaResp.GetError() != nil {
		e := schemaResp.GetError()
		return nil, nil, errors.New(fmt.Sprintf("%d - %s\n%s", e.Code, e.Message, e.Details))
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(schemaResp.GetSchema(), pbSet); err != nil {
		return nil, nil, err
	}

	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		return nil, nil, err
	}

	return reg, pbSet, nil
}
