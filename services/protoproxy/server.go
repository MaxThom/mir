package protoproxy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type ProtoProxyServer struct{}

var uploadMetric prometheus.Counter

func RegisterMetrics(reg prometheus.Registerer) {
	uploadMetric := metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	reg.Register(uploadMetric)
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
