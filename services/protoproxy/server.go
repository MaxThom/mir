package protoproxy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/libs/api/metrics"
	protostore "github.com/maxthom/mir/libs/proto/store"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type ProtoProxyServer struct {
	cons        jetstream.Consumer
	writer      api.WriteAPI
	marshallers *Marhshallers
}

type ProtoPayload struct {
	key  MarshallerKey
	data []byte
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

// TODO offer both NewServer which accepting external dependencies as var and NewServerWithConnections
func NewProtoProxyServer(logger zerolog.Logger, registry *protostore.Registry, cons jetstream.Consumer, writer api.WriteAPI) *ProtoProxyServer {
	l = logger.With().Str("srv", "protoproxy_server").Logger()
	return &ProtoProxyServer{
		cons:        cons,
		writer:      writer,
		marshallers: NewMarshallers(registry),
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (p *ProtoProxyServer) ListenAndPushTelemetry(ctx context.Context) {
	// Optimize with prefetch and batches
	// Optimize with better ack patterns
	// Use iterator pattern instead of Consume
	// Look for flush

	// p.marshallers.Registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
	// 	fmt.Println("Messages in the .proto file:")
	// 	for i := 0; i < fd.Messages().Len(); i++ {
	// 		fmt.Println("-", fd.Messages().Get(i).FullName())
	// 	}
	// 	return true
	// })

	// Create routine for deserializing and writing to db
	dataCh := make(chan ProtoPayload)
	go func(stream <-chan ProtoPayload) {
		for pr := range stream {
			lp, err := p.marshallers.Deserialize(pr.data, pr.key)
			if err != nil {
				l.Error().Err(err).Msg("error while marshalling")
				continue
			}
			// fmt.Println(lp)
			// This is also another channel and routine to write to db
			p.writer.WriteRecord(lp)
		}
	}(dataCh)

	l.Info().Msg("listening to telemetry")
	select {
	case <-ctx.Done():
		l.Info().Msg("shutting down")
		return
	default:
		for {
			// startTime := time.Now()
			msgs, err := p.cons.Fetch(100, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				l.Error().Err(err).Msg("")
			}

			// duration := time.Since(startTime)
			// fmt.Println("duration: ", duration)

			// Extract proto message name and device id to deserialize
			for msg := range msgs.Messages() {
				msgName, ok := msg.Headers()["__pb"]
				if !ok {
					l.Warn().Err(err).Msg("missing proto header")
					continue
				}
				deviceId := ""
				deviceIdAr, ok := msg.Headers()["deviceId"]
				if ok {
					deviceId = deviceIdAr[0]
				}

				dataCh <- ProtoPayload{
					data: msg.Data(),
					key: MarshallerKey{
						messageName: msgName[0],
						deviceId:    deviceId,
					},
				}

				msg.Ack()
				// TODO Adjust to reduce observability cost
				datapointCount.Inc()
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
