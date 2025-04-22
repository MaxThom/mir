package eventstore_srv

import (
	"context"
	"sync"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/services/schema_cache"
	eventstore_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/eventstore_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type EventStoreServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	devStore  mng.MirStore
	schStore  *schema_cache.MirProtoCache
}

const (
	ServiceName = "mir_eventstore"
)

var (
	requestTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "request_total",
		Help:      "Number of request for event store routes",
	}, []string{"route"})
	requestErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "request_error_total",
		Help:      "Number of error request for event store routes",
	}, []string{"route"})
	eventCaptureTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "event_capture_total",
		Help:      "Number of events captured by the system",
	})
	eventCaptureErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "event_capture_error_total",
		Help:      "Number of events captured by the system in error",
	})

	l zerolog.Logger
)

func init() {
	requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
}

func NewEventStore(logger zerolog.Logger, m *mir.Mir, store mng.MirStore) (*EventStoreServer, error) {
	l = logger.With().Str("srv", "eventstore_server").Logger()
	ctx, cancelFn := context.WithCancel(context.Background())
	return &EventStoreServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		wg:        &sync.WaitGroup{},
		m:         m,
		devStore:  store,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *EventStoreServer) Serve() error {
	if err := s.m.Server().ListEvents().QueueSubscribe(ServiceName, s.listEventsSub); err != nil {
		return err
	}
	return nil
}

func (s *EventStoreServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *EventStoreServer) listEventsSub(msg *mir.Msg, clientId string, req *eventstore_apiv1.SendListEventsRequest) ([]*eventstore_apiv1.Event, error) {
	l.Info().Any("req", req).Msg("list events request")
	requestTotal.WithLabelValues("list").Inc()

	l.Info().Msg("list events request processed successfully")
	return []*eventstore_apiv1.Event{}, nil
}
