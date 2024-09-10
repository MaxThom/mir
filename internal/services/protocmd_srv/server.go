package protocmd_srv

import (
	"context"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type ProtoCmdServer struct {
	sub      *nats.Subscription
	m        *mir.Mir
	devStore mng.DeviceStore
	// devWriters     map[deviceProtoKey]proto_lineprotocol.ProtoBytesToLpFn
	// devWritersLock sync.RWMutex
	// devSchemas     map[string]deviceProtoSchema
	// devSchemasLock sync.RWMutex
}

// TODO prom metics
// - number of device schema fetch

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

func NewProtoCmdServer(logger zerolog.Logger, m *mir.Mir, devStore mng.DeviceStore) *ProtoCmdServer {
	l = logger.With().Str("srv", "protoflux_server").Logger()
	return &ProtoCmdServer{
		m:        m,
		devStore: devStore,
	}
}

func (s *ProtoCmdServer) Listen(ctx context.Context) {
	s.m.Subscribe(mir.Stream().V1Alpha().Telemetry(
		func(msg *nats.Msg, deviceId string, protoMsgName string) {
		}))
}
