package eventstore_srv

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/services/schema_cache"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type EventStoreServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	store     mng.MirStore
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
		store:     store,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *EventStoreServer) Serve() error {
	if err := s.m.Server().ListEvents().QueueSubscribe(ServiceName, s.listEventsSub); err != nil {
		return err
	}
	if err := s.m.Server().DeleteEvents().QueueSubscribe(ServiceName, s.deleteEventsSub); err != nil {
		return err
	}
	if err := s.m.Event().QueueSubscribe(ServiceName, s.streamEventsSub); err != nil {
		return err
	}
	return nil
}

func (s *EventStoreServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *EventStoreServer) listEventsSub(msg *mir.Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error) {
	l.Info().Any("req", req).Msg("list events request")
	requestTotal.WithLabelValues("list").Inc()

	respDb, err := s.store.ListEvent(req)
	if err != nil {
		requestErrorTotal.WithLabelValues("list").Inc()
		l.Error().Err(err).Msg("error occure while listing events in a db query")
		return nil, fmt.Errorf("error listing events: %w", err)
	}

	l.Info().Msg("list events request processed successfully")
	return respDb, nil
}

func (s *EventStoreServer) deleteEventsSub(msg *mir.Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error) {
	l.Info().Any("req", req).Msg("delete events request")
	requestTotal.WithLabelValues("delete").Inc()

	evList, err := s.store.DeleteEvent(req)
	if err != nil {
		if errors.Is(err, mir_v1.ErrorNoDeviceTargetProvided) {
			requestErrorTotal.WithLabelValues("delete").Inc()
			return nil, fmt.Errorf("error no target found: %w", err)
		}
		l.Error().Err(err).Msg("error occure while executing delete event request")
		return nil, fmt.Errorf("error deleting event: %w", err)
	}

	return evList, nil
}

func (s *EventStoreServer) streamEventsSub(msg *mir.Msg, subjectId string, req mir_v1.EventSpec, e error) {
	l.Info().Any("req", req).Str("subject", msg.Subject).Msg("event received")
	eventCaptureTotal.Inc()
	defer msg.Ack()
	if e != nil {
		eventCaptureErrorTotal.Inc()
		l.Error().Err(e).Msg("error occure while streaming event")
		return
	}

	// TODO name of events, tbd
	event := mir_v1.NewEvent()
	id, err := uuid.NewV7()
	if err != nil {
		l.Error().Err(err).Msg("error generating UUID")
		return
	}
	if req.RelatedObject.Meta.Name != "" {
		event.Meta.Name = req.RelatedObject.Meta.Name + "-" + id.String()[24:]
	} else {
		event.Meta.Name = id.String()
	}
	event.Meta.Namespace = req.RelatedObject.Meta.Namespace
	if event.Meta.Namespace == "" {
		event.Meta.Namespace = "default"
	}
	if event.Meta.Labels == nil {
		event.Meta.Labels = make(map[string]string)
	}
	if event.Meta.Annotations == nil {
		event.Meta.Annotations = make(map[string]string)
	}
	event.Meta.Annotations[mir.HeaderRoute] = msg.Subject
	event.Meta.Annotations[mir.HeaderSubject] = subjectId
	if len(msg.Header.Values(mir.HeaderTrigger)) > 0 {
		event.Meta.Annotations[mir.HeaderTrigger] = strings.Join(msg.Header.Values(mir.HeaderTrigger), ",")
	}
	event.Meta.Labels["reason"] = req.Reason
	event.Meta.Labels["type"] = req.Type

	event.Spec = req

	// TODO stack algo
	event.Status.Count = 1
	event.Status.FirstAt = time.Now().UTC()
	event.Status.LastAt = time.Now().UTC()

	_, err = s.store.CreateEvent(event)
	if err != nil {
		eventCaptureErrorTotal.Inc()
		l.Error().Err(err).Msg("error occure while streaming event")
		return
	}

	l.Info().Msg("event streamed successfully")
}
