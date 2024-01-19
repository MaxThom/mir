package protoproxy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type ProtoProxyServer struct {
	cons   jetstream.Consumer
	writer api.WriteAPI
}

var uploadMetric prometheus.Counter

func RegisterMetrics(reg prometheus.Registerer) {
	uploadMetric := metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	reg.Register(uploadMetric)
}

func NewProtoProxyServer(cons jetstream.Consumer, writer api.WriteAPI) *ProtoProxyServer {
	return &ProtoProxyServer{
		cons:   cons,
		writer: writer,
	}
}

func (p *ProtoProxyServer) ListenAndPushTelemetry(ctx context.Context) {
	// Optimize with prefetch and batches
	// Optimize with better ack patterns
	// Use iterator pattern instead of Consume
	// Look for flush

	for {
		msgs, err := p.cons.Fetch(100, jetstream.FetchMaxWait(1*time.Second))
		if err != nil {
			fmt.Println(err)
		}
		for msg := range msgs.Messages() {
			fmt.Println(string(msg.Data()))
			p.writer.WriteRecord("")
			msg.Ack()
		}
		if msgs.Error() != nil {
			fmt.Println("Error fetching messages: ", err)
		}
	}

	cc, err := p.cons.Consume(func(msg jetstream.Msg) {
		// TODO
		// deserialize from protobuf to line protocol
		// send to databases
		fmt.Println("Received jetstream message: ", string(msg.Data()))

		p.writer.WriteRecord("")

		msg.Ack()
	})
	if err != nil {
		fmt.Println(err)
	}

	defer cc.Stop()
}

func (p *ProtoProxyServer) UploadSchema(ctx context.Context,
	req *connect.Request[protoproxy.UploadSchemaRequest],
) (*connect.Response[protoproxy.UploadSchemaResponse], error) {
	uploadMetric.Inc()
	log.Info().Msg("upload schema!")
	log.Info().Msg(fmt.Sprintf("Request headers: %s", req.Header()))
	res := connect.NewResponse(&protoproxy.UploadSchemaResponse{
		Msg: "schema uploaded!",
	})
	res.Header().Set("Content-Type", "application/json")
	return res, nil
}
