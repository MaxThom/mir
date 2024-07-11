package protoflux

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
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
}
