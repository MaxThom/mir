package protoproxy

import (
	"context"
	"fmt"
	"os"
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

	dataCh := make(chan string)
	go func(stream <-chan string) {
		for lp := range stream {
			fmt.Fprintf(os.Stdout, "Processing value: %s\n", lp)
			p.writer.WriteRecord("")
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
				dataCh <- string(msg.Data())
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
