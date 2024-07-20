package protoflux

import (
	"context"
	"fmt"
	"log"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/device"
	"github.com/maxthom/mir/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/libs/proto/line_protocol"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoFluxServer struct {
	writer api.WriteAPI
	sub    *nats.Subscription
	m      *mir.Mir
	db     *surrealdb.DB
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

func NewProtoFluxServer(logger zerolog.Logger, m *mir.Mir, writer api.WriteAPI, db *surrealdb.DB) *ProtoFluxServer {
	l = logger.With().Str("srv", "protoflux_server").Logger()
	return &ProtoFluxServer{
		writer: writer,
		m:      m,
		db:     db,
	}
}

func (s *ProtoFluxServer) Listen(ctx context.Context) {
	s.m.Subscribe(mir.Stream().V1Alpha().Telemetry(
		func(msg *nats.Msg, deviceId string, protoMsgName string) {
			fmt.Print("received data > ")
			schemaResp := &device.SchemaRetrieveResponse{}
			err := s.m.SendRequest(mir.Device().V1Alpha().RequestSchema(deviceId, schemaResp))
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
			s.writer.WriteRecord(lp)

		}))
}
