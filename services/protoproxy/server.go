package protoproxy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/libs/proto/line_protocol"
	protostore "github.com/maxthom/mir/libs/proto/store"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoProxyServer struct {
	cons     jetstream.Consumer
	writer   api.WriteAPI
	registry *protostore.Registry
}

var uploadMetric prometheus.Counter

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	uploadMetric := metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	reg.Register(uploadMetric)
}

// TODO offer both NewServer which accepting external dependencies as var and NewServerWithConnections
func NewProtoProxyServer(logger zerolog.Logger, registry *protostore.Registry, cons jetstream.Consumer, writer api.WriteAPI) *ProtoProxyServer {
	l = logger.With().Str("srv", "protoproxy_server").Logger()
	return &ProtoProxyServer{
		cons:     cons,
		writer:   writer,
		registry: registry,
	}
}

func (p *ProtoProxyServer) ListenAndPushTelemetry(ctx context.Context) {
	// Optimize with prefetch and batches
	// Optimize with better ack patterns
	// Use iterator pattern instead of Consume
	// Look for flush

	p.registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fmt.Println("Messages in the .proto file:")

		for i := 0; i < fd.Messages().Len(); i++ {
			fmt.Println("-", fd.Messages().Get(i).FullName())
		}

		return true
	})

	dataCh := make(chan string)
	go func(stream <-chan string) {
		for lp := range stream {
			fmt.Println(lp)
			p.writer.WriteRecord(lp)
			// Add processing logic here
		}
	}(dataCh)

	l.Info().Msg("listening to telemetry")
	select {
	case <-ctx.Done():
		l.Info().Msg("shuting down")
		return
	default:
		for {
			msgs, err := p.cons.Fetch(100, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				l.Error().Err(err).Msg("")
			}
			for msg := range msgs.Messages() {
				fmt.Println(string(msg.Data()))

				fmt.Println(msg.Headers()["__pb"])
				// TODO third channel for processing and ack
				// could be a protoproxy library
				// lazy loading with a hashmap on the descriptors
				protoMsg := msg.Headers()["__pb"][0]
				desc, err := p.registry.FindDescriptorByName(protoreflect.FullName(protoMsg))
				if err != nil {
					l.Error().Err(err).Msg("error while loading descriptor")
				}
				fn, err := proto_lineprotocol.GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
				if err != nil {
					l.Error().Err(err).Msg("error while loading descriptor")
				}

				lp := fn(msg.Data(), map[string]string{})
				dataCh <- string(lp)
				msg.Ack()
			}
			if msgs.Error() != nil {
				l.Error().Err(err).Msg("error while fetching from bus")
			}
		}
	}
}

func (p *ProtoProxyServer) UploadSchema(ctx context.Context,
	req *connect.Request[protoproxy.UploadSchemaRequest],
) (*connect.Response[protoproxy.UploadSchemaResponse], error) {
	uploadMetric.Inc()
	l.Info().Msg("upload schema!")
	l.Info().Msg(fmt.Sprintf("Request headers: %s", req.Header()))
	res := connect.NewResponse(&protoproxy.UploadSchemaResponse{
		Msg: "schema uploaded!",
	})
	res.Header().Set("Content-Type", "application/json")
	return res, nil
}
