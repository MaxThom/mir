package protocfg_srv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/services/schema_cache"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type ProtoCfgServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	devStore  mng.DeviceStore
	schStore  *schema_cache.MirProtoCache
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
	cc, err := schema_cache.NewMirProtoCache(l, m)
	if err != nil {
		return nil, err
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	return &ProtoCfgServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		wg:        &sync.WaitGroup{},
		m:         m,
		devStore:  store,
		schStore:  cc,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *ProtoCfgServer) Serve() error {
	if err := s.m.Server().ListConfiguration().QueueSubscribe(ServiceName, s.listCfgSub); err != nil {
		return err
	}
	return nil
}

func (s *ProtoCfgServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *ProtoCfgServer) listCfgSub(msg *mir.Msg, clientId string, req *cfg_apiv1.SendListConfigRequest) (map[string]*cfg_apiv1.Configs, error) {
	l.Info().Any("req", req).Msg("list config request")
	// 1. get device list
	// 2. for each device, get stored schema, if empty, fetch from device
	// 3. return list of commands

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
		return nil, fmt.Errorf("error listing devices from db: %w", err)
	}

	devsCmds := make(map[string]*cfg_apiv1.Configs)
	for _, dev := range devs {
		reg, _, err := s.schStore.GetDeviceSchema(dev.Spec.DeviceId, req.RefreshSchema)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
				Error: err.Error(),
			}
			continue
		}

		cfgs, err := reg.GetConfigList(req.FilterLabels)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
				Error: err.Error(),
			}
			continue
		}

		cfgList := []*cfg_apiv1.ConfigDescriptor{}
		for _, cmd := range cfgs {
			cfgList = append(cfgList, cmd)
		}
		devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
			Configs: cfgList,
		}
	}

	l.Info().Msg("list command request processed successfully")
	return devsCmds, nil
}
