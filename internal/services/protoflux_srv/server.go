package protoflux_srv

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/device_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoFluxServer struct {
	tlmStore       ts.TelemetryStore
	sub            *nats.Subscription
	m              *mir.Mir
	db             *surrealdb.DB
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

func NewProtoFluxServer(logger zerolog.Logger, m *mir.Mir, tlmStore ts.TelemetryStore, db *surrealdb.DB) *ProtoFluxServer {
	l = logger.With().Str("srv", "protoflux_server").Logger()
	return &ProtoFluxServer{
		tlmStore:   tlmStore,
		m:          m,
		db:         db,
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
				// TODO put all this if inside a function
				s.devSchemasLock.RLock()
				devSchema, ok := s.devSchemas[deviceId]
				s.devSchemasLock.RUnlock()
				// No schema, thus we must check in db first
				// if not found, we must request it from device
				if !ok {
					// TODO look for schema in db first
					// 1. Go get schema in surrealdb
					//    Should be under status
					// status>telemetry:
					//   protoPackageName: "mir.v1alpha.telemetry"
					//   compressedSchema: "..."
					//   lastSchemaFetch: "..."
					// 2. If not there, fetch from device

					l.Info().Str("deviceId", deviceId).Msg("Requesting proto schema from device")
					reg, _, err := getProtoSchemaFromDevice(s.m, deviceId)
					if err != nil {
						l.Error().Err(err).Str("deviceId", devSchema.deviceId).Msg("Failed to retrieve schema from device")
						if err := msg.Nak(); err != nil {
							l.Error().Err(err).Str("deviceId", devSchema.deviceId).Msg("Failed to NAK message of device")
						}
						return
					}
					s.devSchemasLock.Lock()
					devSchema = deviceProtoSchema{
						deviceId: deviceId,
						schema:   reg,
					}
					s.devSchemas[deviceId] = devSchema
					s.devSchemasLock.Unlock()

					// TODO update the schema in db with info
					// status>telemetry:
					//   protoPackageName: "mir.v1alpha.telemetry"
					//   compressedSchema: "..."
					//   lastSchemaFetch: "..."
				}
				fn, err := generateIngesters(devSchema, protoMsgName)
				if err != nil {
					l.Error().Err(err).Str("deviceId", devSchema.deviceId).Str("protoMsg", protoMsgName).Msg("Failed to generate ingester function")
					if err := msg.Nak(); err != nil {
						l.Error().Err(err).Str("deviceId", devSchema.deviceId).Msg("Failed to NAK message of device")
					}
					return
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

func (s *ProtoFluxServer) listenPlayground(ctx context.Context) {
	s.m.Subscribe(mir.Stream().V1Alpha().Telemetry(
		func(msg *nats.Msg, deviceId string, protoMsgName string) {
			fmt.Println("received data > ")
			schemaResp := &device_api.SchemaRetrieveResponse{}
			err := s.m.SendRequest(mir.Command().V1Alpha().RequestSchema(deviceId, schemaResp))
			if err != nil {
				fmt.Println(err)
			} else if schemaResp.GetError() != nil {
				fmt.Println(schemaResp.GetError())
			}

			pbSet := new(descriptorpb.FileDescriptorSet)
			if err := proto.Unmarshal(schemaResp.GetSchema(), pbSet); err != nil {
				log.Fatalf("Failed to unmarshal descriptor: %v", err)
			}

			// Create registry from the FileDescriptorSet
			reg, err := protodesc.NewFiles(pbSet)
			if err != nil {
				log.Fatalf("Failed to create registry: %v", err)
			}

			// Find the descriptor by name
			desc, err := reg.FindDescriptorByName(protoreflect.FullName(protoMsgName))
			if err != nil {
				log.Fatalf("Failed to find descriptor: %v", err)
			}

			// Unmarshal data with the descriptor
			msgType := desc.(protoreflect.MessageDescriptor)
			dynMsg := dynamicpb.NewMessage(msgType)
			if err := proto.Unmarshal(msg.Data, dynMsg); err != nil {
				log.Fatalf("Failed to deserialize message: %v", err)
			}
			fmt.Println(dynMsg)
			b, err := protojson.Marshal(dynMsg.Interface())
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(string(b))
			fmt.Println("schema size ", len(schemaResp.GetSchema()))
			fmt.Println("datapoint size ", len(msg.Data))
			fmt.Println("json dp size ", len(b))

			// Write data to influxdb
			fn, err := proto_lineprotocol.GenerateMarshalFn(map[string]string{
				"deviceId":  deviceId,
				"namespace": "mir", // TODO find device namespace
			}, desc.(protoreflect.MessageDescriptor))
			if err != nil {
				log.Fatalf("Failed to generate marshal function: %v", err)
			}

			lp := fn(msg.Data, map[string]string{})
			fmt.Println(lp)
			s.tlmStore.WriteDatapoint(lp)
		}))
}
