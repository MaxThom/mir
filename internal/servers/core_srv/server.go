package core_srv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/retry"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
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

type CoreServer struct {
	ctx                      context.Context
	cancelCtx                context.CancelFunc
	wg                       *sync.WaitGroup
	m                        *mir.Mir
	store                    mng.MirStore
	hearthbeats              map[mir_v1.DeviceId]time.Time
	hearthbeatsMutex         sync.RWMutex
	hearthbeatsWriteBuffer   map[mir_v1.DeviceId]mir_v1.DeviceHello
	hearthbeatsWriteBufferMu sync.RWMutex
	opts                     Options
}

type Options struct {
	DeviceOnlineFlush  time.Duration
	DeviceOfflineFlush time.Duration
	DeviceOfflineAfter time.Duration
}

const (
	ServiceName = "mir_core"
)

var (
	requestTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "core",
		Name:      "request_total",
		Help:      "Number of request for core",
	}, []string{"route"})
	requestErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "core",
		Name:      "request_error_total",
		Help:      "Number of error request for core",
	}, []string{"route"})
	deviceStatusCount = metrics.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "core",
		Name:      "device_status_count",
		Help:      "Number of devices online or offline",
	}, []string{"status"})

	l zerolog.Logger
)

func init() {
	requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestTotal.With(prometheus.Labels{"route": "create"}).Add(0)
	requestTotal.With(prometheus.Labels{"route": "update"}).Add(0)
	requestTotal.With(prometheus.Labels{"route": "delete"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "create"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "update"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "delete"}).Add(0)
	deviceStatusCount.With(prometheus.Labels{"status": "online"}).Add(0)
	deviceStatusCount.With(prometheus.Labels{"status": "offline"}).Add(0)
}

func NewCore(logger zerolog.Logger, m *mir.Mir, store mng.MirStore, opts *Options) (*CoreServer, error) {
	l = logger.With().Str("srv", "core_server").Logger()
	if opts == nil {
		opts = &Options{
			DeviceOnlineFlush:  7 * time.Second,
			DeviceOfflineFlush: 12 * time.Second,
			DeviceOfflineAfter: 30 * time.Second,
		}
	}
	if opts.DeviceOnlineFlush == 0 {
		opts.DeviceOnlineFlush = 7 * time.Second
	}
	if opts.DeviceOfflineFlush == 0 {
		opts.DeviceOfflineFlush = 12 * time.Second
	}
	if opts.DeviceOfflineAfter == 0 {
		opts.DeviceOfflineAfter = 30 * time.Second
	}

	hearbeats := map[mir_v1.DeviceId]time.Time{}

	// Preload hearthbeat map. Required in case the
	// app is down while a device is also, but report as online
	// because it went offline when the app was down
	devices, err := store.ListDevice(mir_v1.DeviceTarget{}, false)
	if err != nil {
		l.Error().Err(err).Msg("error occure while executing list query")
	}
	timeNow := time.Now().UTC()
	for _, d := range devices {
		// We only add the online ones
		// This way, the pulse doesnt do a check on offline device
		// When an offline device sends a first pulse, it get added
		// to the map.
		// If a device becomes offline, it's removed from the map
		if d.Status.Online != nil && *d.Status.Online {
			// Edge case, device will be set to offline on next pulsor offline cycle
			if d.Status.LastHearthbeat == nil {
				hearbeats[mir_v1.DeviceId(d.Spec.DeviceId)] = timeNow
			} else {
				hearbeats[mir_v1.DeviceId(d.Spec.DeviceId)] = d.Status.LastHearthbeat.Time
			}
			deviceStatusCount.WithLabelValues("online").Inc()
		} else {
			deviceStatusCount.WithLabelValues("offline").Inc()
		}
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	return &CoreServer{
		ctx:                    ctx,
		cancelCtx:              cancelFn,
		wg:                     &sync.WaitGroup{},
		m:                      m,
		store:                  store,
		hearthbeats:            hearbeats,
		hearthbeatsWriteBuffer: make(map[mir_v1.DeviceId]mir_v1.DeviceHello),
		opts:                   *opts,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *CoreServer) Serve() error {
	if err := s.m.Client().CreateDevice().QueueSubscribe(ServiceName, s.createDeviceSub); err != nil {
		return err
	}
	if err := s.m.Client().CreateDevices().QueueSubscribe(ServiceName, s.createDevicesSub); err != nil {
		return err
	}
	if err := s.m.Client().UpdateDevice().QueueSubscribe(ServiceName, s.updateDeviceSub); err != nil {
		return err
	}
	if err := s.m.Client().DeleteDevice().QueueSubscribe(ServiceName, s.deleteDeviceSub); err != nil {
		return err
	}
	if err := s.m.Client().ListDevice().QueueSubscribe(ServiceName, s.listDeviceSub); err != nil {
		return err
	}
	if err := s.m.Device().Hearthbeat().QueueSubscribe(ServiceName, "*", s.hearthbeatSub); err != nil {
		return err
	}
	if err := s.m.Device().Schema().QueueSubscribe(ServiceName, "*", s.schemaSub); err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		offlineTicker := time.NewTicker(s.opts.DeviceOfflineFlush)
		onlineTicker := time.NewTicker(s.opts.DeviceOnlineFlush)
		defer offlineTicker.Stop()
		defer onlineTicker.Stop()
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				l.Debug().Msg("shutting down pulsor task")
				return
			case <-offlineTicker.C:
				s.hearthbeatOfflinePulsor(s.opts.DeviceOfflineAfter)
			case <-onlineTicker.C:
				s.hearthbeatOnlinePulsor()
			}
		}
	}()
	return nil
}

func (s *CoreServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *CoreServer) createDeviceSub(msg *mir.Msg, clientId string, d mir_v1.Device) (mir_v1.Device, error) {
	l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", d)).Msg("new device request")
	requestTotal.WithLabelValues("create").Inc()

	newDev, err := s.store.CreateDevice(d)
	if err != nil {
		l.Error().Err(err).Msg("error occure while creating device")
		requestErrorTotal.WithLabelValues("create").Inc()
	}

	// Publish created events
	if err := publishDeviceCreateEvent(s.m, msg, newDev); err != nil {
		l.Warn().Err(err).Str("device_id", newDev.Spec.DeviceId).Msg("error occure while publishing device created event")
	}

	return newDev, err
}

func (s *CoreServer) createDevicesSub(msg *mir.Msg, clientId string, d []mir_v1.Device) ([]mir_v1.Device, error) {
	l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", d)).Msg("new devices request")
	requestTotal.WithLabelValues("create").Inc()

	newDevs, err := s.store.CreateDevices(d)
	if err != nil {
		l.Error().Err(err).Msg("error occure while creating devices")
		requestErrorTotal.WithLabelValues("create").Inc()
		return nil, fmt.Errorf("error creating devices: %w", err)
	}

	// Publish created events
	for _, newDev := range newDevs {
		if err := publishDeviceCreateEvent(s.m, msg, newDev); err != nil {
			l.Warn().Err(err).Str("device_id", newDev.Spec.DeviceId).Msg("error occure while publishing device created event")
		}
	}

	return newDevs, err
}

func (s *CoreServer) updateDeviceSub(msg *mir.Msg, clientId string, t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error) {
	l.Debug().Str("route", "update").Str("payload", fmt.Sprintf("%v", t)).Msg("update device request")
	requestTotal.WithLabelValues("update").Inc()
	// Send config to cfg module
	// We do it twice, one with dry run to validate the config
	// It seems slow and redundant, but we must validate the config for all
	if len(d.Properties.Desired) > 0 {
		l.Debug().Str("route", "update").Msg("sending config to cfg module")
		// props := req.GetProps().GetDesired().Fields

		// First we validate all the properties
		// We do this as we do not want to have only a subset of the request to be written
		var errs error
		for k, v := range d.Properties.Desired {
			cfgRespDryRun, err := s.m.Client().SendConfig().RequestJson(&mir.SendDeviceConfigRequestJson{
				Targets:        mir_v1.MirDeviceTargetToProtoDeviceTarget(t),
				CommandName:    k,
				CommandPayload: v,
				DryRun:         true,
			})
			if err != nil {
				l.Error().Err(err).Msgf("error validating config '%s' to cfg module", k)
				errs = errors.Join(fmt.Errorf("error validating config '%s' to cfg module: %w", k, err))
			}
			for devName, cfg := range cfgRespDryRun {
				if cfg.Error != "" {
					l.Error().Err(err).Msgf("error validating config '%s' to device %s", k, devName)
					errs = errors.Join(fmt.Errorf("error validating config '%s' to device %s: %s", k, devName, cfg.Error))
				}
			}
		}
		if errs != nil {
			requestErrorTotal.WithLabelValues("update").Inc()
			return nil, errs
		}

		// If all validated, we can send
		// We know they were validated, so if error, it means its to the device
		for k, v := range d.Properties.Desired {
			s.m.Client().SendConfig().RequestJson(&mir.SendDeviceConfigRequestJson{
				Targets:           mir_v1.MirDeviceTargetToProtoDeviceTarget(t),
				CommandName:       k,
				CommandPayload:    v,
				SendOnlyDifferent: true,
			})
		}
	}

	respDb, err := s.store.UpdateDevice(t, d)
	if err != nil {
		if errors.Is(err, mng.ErrorDeviceShouldBeCreated) {
			resp, err := s.m.Client().CreateDevice().Request(d)
			if err != nil {
				l.Error().Err(err).Msg("error creating device")
				requestErrorTotal.WithLabelValues("update").Inc()
				return nil, fmt.Errorf("error creating device: %w", err)
			}
			l.Info().Str("route", "device_update").Str("device_id", resp.Spec.DeviceId).Msg("new device created from update (upsert)")
			respDb = []mir_v1.Device{resp}
		} else if errors.Is(err, mir_v1.ErrorNoDeviceTargetProvided) {
			requestErrorTotal.WithLabelValues("update").Inc()
			l.Error().Err(err).Msg("error no target found")
			return nil, fmt.Errorf("error no target found: %w", err)
		} else {
			requestErrorTotal.WithLabelValues("update").Inc()
			l.Error().Err(err).Msg("error updating device")
			return nil, fmt.Errorf("error updating device: %w", err)
		}
	}

	// Publish update events
	for _, d := range respDb {
		if err := publishDeviceUpdateEvent(s.m, msg, d); err != nil {
			l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device updated event")
		}
	}

	return respDb, nil
}

func (s *CoreServer) deleteDeviceSub(msg *mir.Msg, clientId string, t mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
	l.Debug().Str("route", "delete").Str("payload", fmt.Sprintf("%v", t)).Msg("delete device request")
	requestTotal.WithLabelValues("delete").Inc()

	devList, err := s.store.DeleteDevice(t)
	if err != nil {
		if errors.Is(err, mir_v1.ErrorNoDeviceTargetProvided) {
			requestErrorTotal.WithLabelValues("delete").Inc()
			return nil, fmt.Errorf("error no target found: %w", err)
		}
		l.Error().Err(err).Msg("error occure while executing delete device request")
		return nil, fmt.Errorf("error deleting device: %w", err)
	}

	// Publish delete events
	for _, d := range devList {
		if err := publishDeviceDeleteEvent(s.m, msg, d); err != nil {
			l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device deleted event")
		}
	}
	return devList, nil
}

func (s *CoreServer) listDeviceSub(msg *mir.Msg, clientId string, t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
	l.Debug().Str("client_id", clientId).Str("route", "list").Str("payload", fmt.Sprintf("%v", t)).Msg("list device request")
	requestTotal.WithLabelValues("list").Inc()

	respDb, err := s.store.ListDevice(t, includeEvents)
	if err != nil {
		requestErrorTotal.WithLabelValues("list").Inc()
		l.Error().Err(err).Msg("error occure while executing a db query")
		return nil, fmt.Errorf("error listing device: %w", err)
	}

	return respDb, nil
}

func (s *CoreServer) hearthbeatOfflinePulsor(offlineAfter time.Duration) {
	l.Trace().Msg("hearthbeats offline pulse")
	s.hearthbeatsMutex.Lock()
	defer s.hearthbeatsMutex.Unlock()
	newOffline := []string{}
	now := time.Now().UTC()
	for k, v := range s.hearthbeats {
		if v.Add(offlineAfter).Before(now) {
			newOffline = append(newOffline, string(k))
		}
	}
	if len(newOffline) > 0 {
		toBoolRef := func(b bool) *bool {
			return &b
		}
		t := mir_v1.DeviceTarget{
			Ids: newOffline,
		}
		d := mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
			Online: toBoolRef(false),
		})
		devs, err := s.store.UpdateDevice(t, d)
		if err != nil {
			if !strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
				l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
			}
			// Degraded mode
			devs = make([]mir_v1.Device, len(newOffline))
			for i, id := range newOffline {
				devs[i] = mir_v1.NewDevice().WithId(id).WithMeta(mir_v1.Meta{
					Name:      id,
					Namespace: "__degraded",
				})
			}
		}
		l.Info().Str("route", "hearthbeat_pulsor").Str("event", "device_offline").Strs("new devices", newOffline).Msg("offline devices")
		for _, d := range devs {
			if err := publishDeviceOfflineEvent(s.m, nil, d); err != nil {
				l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device offline event")
			}
			delete(s.hearthbeats, mir_v1.DeviceId(d.Spec.DeviceId))
		}
		deviceStatusCount.WithLabelValues("offline").Add(float64(len(devs)))
		deviceStatusCount.WithLabelValues("online").Sub(float64(len(devs)))

		l.Info().Strs("new_offline_devices", newOffline).Msg("hearthbeats offline pulse")
	}
}

func (s *CoreServer) hearthbeatOnlinePulsor() {
	l.Trace().Msg("hearthbeats online pulse")
	s.hearthbeatsWriteBufferMu.Lock()
	tempBuffer := maps.Clone(s.hearthbeatsWriteBuffer)
	s.hearthbeatsWriteBuffer = make(map[mir_v1.DeviceId]mir_v1.DeviceHello, len(s.hearthbeatsWriteBuffer))
	s.hearthbeatsWriteBufferMu.Unlock()

	degradedMode := false
	devsResp, err := retry.RetryOnErrorContainsWithResult(func() ([]mir_v1.Device, error) {
		res, err := s.store.UpdateDeviceHello(tempBuffer)
		return res, err
	}, "Failed to commit transaction due to a read or write conflict.", 10, 100*time.Millisecond)
	if err != nil {
		if !strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
			l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
		}
		// Degraded mode
		degradedMode = true
		devsResp = make([]mir_v1.Device, len(tempBuffer))
		i := 0
		for deviceId, t := range tempBuffer {
			devsResp[i] = mir_v1.NewDevice().WithId(string(deviceId)).WithMeta(mir_v1.Meta{
				Name:      string(deviceId),
				Namespace: "__degraded",
			}).WithStatus(mir_v1.DeviceStatus{
				Online:         &degradedMode,
				LastHearthbeat: &surrealdbModels.CustomDateTime{Time: t.Hearthbeat},
			}).WithSchema(t.Schema, t.Hearthbeat)
			i += 1
		}
	}

	// devsResp contains all devices that were updated, if devices was not updated,
	// it means it does not exist in db thus we need to provision it
	if len(devsResp) != len(tempBuffer) && !degradedMode {
		updatedDeviceIds := make(map[mir_v1.DeviceId]bool)
		newDevices := make(map[mir_v1.DeviceId]mir_v1.DeviceHello)
		for _, device := range devsResp {
			updatedDeviceIds[mir_v1.DeviceId(device.Spec.DeviceId)] = true
		}
		for id, t := range tempBuffer {
			if !updatedDeviceIds[id] {
				newDevices[id] = t
			}
		}
		newDevs := []mir_v1.Device{}
		newDevsId := []string{}
		for id := range newDevices {
			newDevs = append(newDevs, mir_v1.NewDevice().WithSpec(mir_v1.DeviceSpec{
				DeviceId: string(id),
			}))
			newDevsId = append(newDevsId, string(id))
		}
		_, err := s.m.Client().CreateDevices().Request(newDevs)
		if err != nil {
			l.Error().Err(err).Strs("device_ids", newDevsId).Msg("could not automaticly provision new devices")
		}
		newDevsResp, err := s.store.UpdateDeviceHello(newDevices)
		if err != nil {
			if !strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
				l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
			}
		}
		devsResp = append(devsResp, newDevsResp...)
		deviceStatusCount.WithLabelValues("offline").Add(float64(len(newDevsResp)))
	}

	newOnline := []string{}
	s.hearthbeatsMutex.Lock()
	for _, dev := range devsResp {
		if _, ok := s.hearthbeats[mir_v1.DeviceId(dev.Spec.DeviceId)]; !ok {
			l.Info().Str("route", "hearthbeat").Str("event", "device_online").Msg(dev.Spec.DeviceId)
			if err := publishDeviceOnlineEvent(s.m, nil, dev); err != nil {
				l.Warn().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error occure while publishing device online event")
			}
			deviceStatusCount.WithLabelValues("online").Inc()
			deviceStatusCount.WithLabelValues("offline").Dec()
			newOnline = append(newOnline, dev.Spec.DeviceId)
		}
		s.hearthbeats[mir_v1.DeviceId(dev.Spec.DeviceId)] = dev.Status.LastHearthbeat.Time

		if hello, ok := tempBuffer[mir_v1.DeviceId(dev.Spec.DeviceId)]; ok && hello.Schema != nil {
			if err := publishDeviceUpdateEvent(s.m, nil, dev); err != nil {
				l.Warn().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error occure while publishing device update event")
			}
		}
	}
	s.hearthbeatsMutex.Unlock()

	if len(newOnline) > 0 {
		l.Info().Strs("new_online_devices", newOnline).Msg("new online devices")
	}
}

func (s *CoreServer) hearthbeatSub(msg *mir.Msg, deviceId string, hello mir_v1.DeviceHello, err error) {
	l.Trace().Str("route", "hearthbeat").Msg("hearthbeat device request")
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("upstream error in sdk")
	}

	s.hearthbeatsWriteBufferMu.Lock()
	s.hearthbeatsWriteBuffer[mir_v1.DeviceId(deviceId)] = hello
	s.hearthbeatsWriteBufferMu.Unlock()
	msg.Ack()
}

func (s *CoreServer) schemaSub(msg *mir.Msg, deviceId string, sch *mir_proto.MirProtoSchema, err error) {
	if err != nil {
		l.Error().Err(err).Msg("upstream error in sdk")
		msg.Ack()
		return
	}
	l.Trace().Str("route", "schema").Msg("schema device request")

	compressSch, err := sch.CompressSchema()
	if err != nil {
		l.Error().Err(err).Msg("error compressing schema for store")
		msg.Ack()
		return
	}

	l.Debug().Str("route", "schema").Str("device_id", deviceId).Msg("schema device request")
	timeNow := time.Now().UTC()
	_, err = s.m.Client().UpdateDevice().Request(
		mir_v1.DeviceTarget{
			Ids: []string{deviceId},
		},
		mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
			Schema: mir_v1.Schema{
				CompressedSchema: compressSch,
				PackageNames:     sch.GetPackageList(),
				LastSchemaFetch:  &surrealdbModels.CustomDateTime{Time: timeNow},
			},
		}),
	)
	if err != nil {
		l.Error().Err(err).Msg("error updating schema for store")
		msg.Ack()
	}

	msg.Ack()
}

func publishDeviceOnlineEvent(m *mir.Mir, msg *mir.Msg, d mir_v1.Device) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return m.Event().Publish(m.Event().NewSubjectString(core_client.DeviceOnlineEvent.WithId(d.Spec.DeviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceOnline",
			Message: "Device is now online",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      d.Meta.Name,
				Namespace: d.Meta.Namespace,
			}).Object,
		}, msg)
}

func publishDeviceOfflineEvent(m *mir.Mir, msg *mir.Msg, d mir_v1.Device) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return m.Event().Publish(m.Event().NewSubjectString(core_client.DeviceOfflineEvent.WithId(d.Spec.DeviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceOffline",
			Message: "Device is now offline",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      d.Meta.Name,
				Namespace: d.Meta.Namespace,
			}).Object,
		}, msg)
}

func publishDeviceCreateEvent(m *mir.Mir, msg *mir.Msg, d mir_v1.Device) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return m.Event().Publish(m.Event().NewSubjectString(core_client.DeviceCreatedEvent.WithId(d.Spec.DeviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceCreated",
			Message: "A device has been created successfully",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      d.Meta.Name,
				Namespace: d.Meta.Namespace,
			}).Object,
		}, msg)
}

func publishDeviceUpdateEvent(m *mir.Mir, msg *mir.Msg, d mir_v1.Device) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return m.Event().Publish(m.Event().NewSubjectString(core_client.DeviceUpdatedEvent.WithId(d.Spec.DeviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceUpdated",
			Message: "A device has been updated successfully",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      d.Meta.Name,
				Namespace: d.Meta.Namespace,
			}).Object,
		}, msg)
}

func publishDeviceDeleteEvent(m *mir.Mir, msg *mir.Msg, d mir_v1.Device) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return m.Event().Publish(m.Event().NewSubjectString(core_client.DeviceDeletedEvent.WithId(d.Spec.DeviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceDeleted",
			Message: "A device has been deleted successfully",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      d.Meta.Name,
				Namespace: d.Meta.Namespace,
			}).Object,
		}, msg)
}
