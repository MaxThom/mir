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
	"github.com/maxthom/mir/internal/libs/external"
	"github.com/maxthom/mir/internal/services/schema_cache"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type EventStoreServer struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	wg             *sync.WaitGroup
	m              *mir.Mir
	store          mng.MirStore
	schStore       *schema_cache.MirSchemaCache
	eventsBuffer   []mir_v1.Event
	eventsBufferMu sync.RWMutex
	opts           Options
}

type Options struct {
	FlushInterval time.Duration
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
	eventCaptureTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "event_capture_total",
		Help:      "Number of events captured by the system",
	}, []string{"event"})
	eventCaptureErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "eventstore",
		Name:      "event_capture_error_total",
		Help:      "Number of events captured by the system in error",
	}, []string{"event"})
	eventBufferSize = metrics.NewGauge(prometheus.GaugeOpts{
		Subsystem: "eventstore",
		Name:      "event_buffer_size",
		Help:      "Number of events in buffer",
	})

	l zerolog.Logger
)

func init() {
	requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
}

func NewEventStore(logger zerolog.Logger, m *mir.Mir, store mng.MirStore, opts *Options) (*EventStoreServer, error) {
	l = logger.With().Str("srv", "eventstore_server").Logger()
	ctx, cancelFn := context.WithCancel(context.Background())

	if opts == nil {
		opts = &Options{
			FlushInterval: 10 * time.Second,
		}
	}
	if opts.FlushInterval == 0 {
		opts.FlushInterval = 10 * time.Second
	}

	return &EventStoreServer{
		ctx:            ctx,
		cancelCtx:      cancelFn,
		wg:             &sync.WaitGroup{},
		m:              m,
		store:          store,
		eventsBuffer:   []mir_v1.Event{},
		eventsBufferMu: sync.RWMutex{},
		opts:           *opts,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *EventStoreServer) Serve() error {
	s.wg.Add(1)
	go func() {
		ticker := time.NewTicker(s.opts.FlushInterval)
		defer ticker.Stop()
		defer s.wg.Done()
		for {
			select {
			case <-ticker.C:
				s.flushBuffer()
			case <-s.ctx.Done():
				return
			}
		}
	}()

	s.wg.Add(1)
	go func() {
		ch := s.store.StatusSubscribe()
		for {
			select {
			case status := <-ch:
				s.dbConnUpdate(status)
			case <-s.ctx.Done():
				s.wg.Done()
				return
			}
		}
	}()

	if err := s.m.Client().ListEvents().QueueSubscribe(ServiceName, s.listEventsSub); err != nil {
		return err
	}
	if err := s.m.Client().DeleteEvents().QueueSubscribe(ServiceName, s.deleteEventsSub); err != nil {
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
	l.Trace().Any("req", req).Str("subject", msg.Subject).Msg("event received")
	eventCaptureTotal.WithLabelValues(req.Reason).Inc()
	defer msg.Ack()
	if e != nil {
		eventCaptureErrorTotal.WithLabelValues(req.Reason).Inc()
		l.Error().Err(e).Msg("error occure while streaming event")
		return
	}

	event := mir_v1.NewEvent()
	id, err := uuid.NewV7()
	if err != nil {
		l.Error().Err(err).Msg("error generating UUID")
		return
	}
	if req.RelatedObject.Meta.Name != "" {
		event.Meta.Name = req.RelatedObject.Meta.Name + "-" + id.String()[24:]
	} else {
		event.Meta.Name = subjectId + "-" + id.String()[24:]
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
	now := new(time.Now().UTC())
	event.Status.Count = 1
	event.Status.FirstAt = now
	event.Status.LastAt = now

	if err = s.sendToBuffer(event); err != nil {
		eventCaptureErrorTotal.WithLabelValues(req.Reason).Inc()
		l.Error().Err(err).Msg("error occure while streaming event")
		return
	}
}

func (s *EventStoreServer) sendToBuffer(event mir_v1.Event) error {
	s.eventsBufferMu.Lock()
	defer s.eventsBufferMu.Unlock()
	s.eventsBuffer = append(s.eventsBuffer, event)
	eventBufferSize.Inc()
	return nil
}

func (s *EventStoreServer) sendToDb(event mir_v1.Event) error {
	_, err := s.store.CreateEvent(event)
	if err != nil {
		s.eventsBufferMu.Lock()
		s.eventsBuffer = append(s.eventsBuffer, event)
		eventBufferSize.Inc()
		s.eventsBufferMu.Unlock()
	}
	return err
}

func (s *EventStoreServer) flushBuffer() {
	s.eventsBufferMu.Lock()
	tempBuffer := make([]mir_v1.Event, len(s.eventsBuffer))
	copy(tempBuffer, s.eventsBuffer)
	s.eventsBuffer = []mir_v1.Event{}
	eventBufferSize.Set(0)
	s.eventsBufferMu.Unlock()

	if _, err := s.store.CreateEvents(tempBuffer); err != nil {
		l.Error().Err(err).Int("count", len(tempBuffer)).Msg("error sending events from buffer in batch, re-adding to buffer")
		// Re add unprocessed item to the buffer
		s.eventsBufferMu.Lock()
		s.eventsBuffer = append(s.eventsBuffer, tempBuffer...)
		eventBufferSize.Add(float64(len(tempBuffer)))
		s.eventsBufferMu.Unlock()
	}
	l.Debug().Int("count", len(tempBuffer)).Msg("flushed events from buffer to db")
}

func (s *EventStoreServer) dbConnUpdate(status external.ConnectionStatus) {
	if s.store.Status() == external.StatusConnected {
		l.Info().Str("status", status.String()).Int("buffer count", len(s.eventsBuffer)).Msg("database reconnected: sending buffered events to storage")
	} else {
		l.Warn().Str("status", status.String()).Msg("database disconnected: sending events to buffer")
	}
}
