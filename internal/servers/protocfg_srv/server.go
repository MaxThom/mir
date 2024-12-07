package protocfg_srv

import (
	"context"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// How to make this service scalable
// The problem is with the hearthbeat map, it need to reconcile between the instances
//
// Option 1, remove the map
//  Do a list query on the store for each new hearthbeat
//  Fetch all devices in the pulsor and do the check
//  The number of db request increase by a lot
//
// Option 2, keep the map
//  Listen to device update event and add to map entry if online, or delete entry if offline
//  For pulsor, use distributed locking mechanism
//
// Option 3, distributed keyvalue store
//  Use a distributed keyvalue store like etcd or natskv to store the map

type ProtoCfgServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	store     mng.DeviceStore
}

const (
	ServiceName = "mir_protocfg"
)

var requestCount = metrics.NewCounterVec(prometheus.CounterOpts{
	Name: "request_count",
	Help: "Number of request for core",
}, []string{"route"})

var (
	l            zerolog.Logger
	offlineAfter = time.Second * 30
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(requestCount)
}

func NewProtoCfg(logger zerolog.Logger, m *mir.Mir, store mng.DeviceStore) (*ProtoCfgServer, error) {
	l = logger.With().Str("srv", "protocfg_server").Logger()
	ctx, cancelFn := context.WithCancel(context.Background())
	return &ProtoCfgServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		wg:        &sync.WaitGroup{},
		m:         m,
		store:     store,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *ProtoCfgServer) Serve() error {
	return nil
}

func (s *ProtoCfgServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}
