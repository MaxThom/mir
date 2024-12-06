package core_srv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	ctx              context.Context
	cancelCtx        context.CancelFunc
	wg               *sync.WaitGroup
	m                *mir.Mir
	store            mng.DeviceStore
	hearthbeats      map[string]time.Time
	hearthbeatsMutex sync.RWMutex
}

const (
	ServiceName = "mir_core"
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

func NewCore(logger zerolog.Logger, m *mir.Mir, store mng.DeviceStore) (*CoreServer, error) {
	l = logger.With().Str("srv", "core_server").Logger()
	hearbeats := map[string]time.Time{}

	// Preload hearthbeat map. Required in case the
	// app is down while a device is also, but report as online
	// because it went offline when the app was down
	devices, err := store.ListDevice(&core_apiv1.ListDeviceRequest{Targets: &core_apiv1.Targets{}})
	if err != nil {
		l.Error().Err(err).Msg("error occure while executing list query")
	}
	for _, d := range devices {
		// We only add the online ones
		// This way, the pulse doesnt do a check on offline device
		// When an offline device sends a first pulse, it get added
		// to the map.
		// If a device becomes offline, it's removed from the map
		if d.Status.Online {
			hearbeats[d.Spec.DeviceId] = d.Status.LastHearthbeat
		}
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	return &CoreServer{
		ctx:         ctx,
		cancelCtx:   cancelFn,
		wg:          &sync.WaitGroup{},
		m:           m,
		store:       store,
		hearthbeats: hearbeats,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *CoreServer) Serve() error {
	if err := s.m.Server().CreateDevice().QueueSubscribe(ServiceName, s.createDeviceSub); err != nil {
		return err
	}
	if err := s.m.Server().UpdateDevice().QueueSubscribe(ServiceName, s.updateDeviceSub); err != nil {
		return err
	}
	if err := s.m.Server().DeleteDevice().QueueSubscribe(ServiceName, s.deleteDeviceSub); err != nil {
		return err
	}
	if err := s.m.Server().ListDevice().QueueSubscribe(ServiceName, s.listDeviceSub); err != nil {
		return err
	}
	if err := s.m.Device().Hearthbeat().QueueSubscribe(ServiceName, "*", s.hearthbeatSub); err != nil {
		return err
	}

	s.wg.Add(1)
	go func() {
		s.hearthbeatPulsor(s.ctx, time.Second*10, offlineAfter)
		s.wg.Done()
	}()
	return nil
}

func (s *CoreServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *CoreServer) createDeviceSub(msg *mir.Msg, clientId string, req *core_apiv1.CreateDeviceRequest) (*core_apiv1.Device, error) {
	l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", req)).Msg("new device request")

	newDev, err := s.store.CreateDevice(req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while creating device")
		return nil, fmt.Errorf("error creating device: %w", err)
	}

	// Publish created events
	err = s.m.Event().DeviceCreate().Publish(msg.GetOriginalTriggerId(), newDev)
	if err != nil {
		l.Warn().Err(err).Str("deviceId", newDev.Spec.DeviceId).Msg("error occure while publishing device created event")
	}
	return mir_models.NewProtoDeviceFromDevice(newDev), nil
}

func (s *CoreServer) updateDeviceSub(msg *mir.Msg, clientId string, req *core_apiv1.UpdateDeviceRequest) ([]*core_apiv1.Device, error) {
	l.Debug().Str("route", "update").Str("payload", fmt.Sprintf("%v", req)).Msg("update device request")

	respDb, err := s.store.UpdateDevice(req)
	if err != nil {
		if errors.Is(err, mng.ErrorDeviceShouldBeCreated) {
			resp, err := s.m.Server().CreateDevice().Request(mir_models.NewCreateDeviceReqFromDeviceUpdateRequest(req))
			if err != nil {
				l.Error().Err(err).Msg("error creating device")
				return nil, fmt.Errorf("error creating device: %w", err)
			}
			l.Info().Str("route", "device_update").Str("device_id", resp.Spec.DeviceId).Msg("new device created from update (upsert)")
			respDb = []mir_models.Device{resp}
		} else if errors.Is(err, mir_models.ErrorNoDeviceTargetProvided) {
			l.Error().Err(err).Msg("error no target found")
			return nil, fmt.Errorf("error no target found: %w", err)
		} else {
			l.Error().Err(err).Msg("error updating device")
			return nil, fmt.Errorf("error updating device: %w", err)
		}
	}

	// Publish update events
	for _, d := range respDb {
		err := s.m.Event().DeviceUpdate().Publish(msg.GetOriginalTriggerId(), d)
		if err != nil {
			l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device updated event")
		}
	}

	return mir_models.NewProtoDeviceListFromDevices(respDb), nil
}

func (s *CoreServer) deleteDeviceSub(msg *mir.Msg, clientId string, req *core_apiv1.DeleteDeviceRequest) ([]*core_apiv1.Device, error) {
	l.Debug().Str("route", "delete").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

	devList, err := s.store.DeleteDevice(req)
	if err != nil {
		if errors.Is(err, mir_models.ErrorNoDeviceTargetProvided) {
			return nil, fmt.Errorf("error no target found: %w", err)
		}
		l.Error().Err(err).Msg("error occure while executing delete device request")
		return nil, fmt.Errorf("error deleting device: %w", err)
	}

	// Publish delete events
	for _, d := range devList {
		err := s.m.Event().DeviceDelete().Publish(msg.GetOriginalTriggerId(), d)
		if err != nil {
			l.Warn().Err(err).Str("deviceId", d.Spec.DeviceId).Msg("error occure while publishing device deleted event")
		}
	}
	return mir_models.NewProtoDeviceListFromDevices(devList), nil
}

func (s *CoreServer) listDeviceSub(msg *mir.Msg, clientId string, req *core_apiv1.ListDeviceRequest) ([]*core_apiv1.Device, error) {
	l.Debug().Str("route", "list").Str("payload", fmt.Sprintf("%v", req)).Msg("list device request")

	respDb, err := s.store.ListDevice(req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while executing a db query")
		return nil, fmt.Errorf("error listing device: %w", err)
	}

	return mir_models.NewProtoDeviceListFromDevices(respDb), nil
}

func (s *CoreServer) hearthbeatPulsor(ctx context.Context, interval time.Duration, offlineAfter time.Duration) {
	for {
		select {
		case <-ctx.Done():
			l.Debug().Msg("shutting down pulsor task")
			return
		case <-time.After(interval):
			s.hearthbeatsMutex.RLock()
			newOffline := []string{}
			now := time.Now().UTC()
			for k, v := range s.hearthbeats {
				if v.Add(offlineAfter).Before(now) {
					newOffline = append(newOffline, k)
				}
			}
			s.hearthbeatsMutex.RUnlock()
			if len(newOffline) > 0 {
				toBoolRef := func(b bool) *bool {
					return &b
				}
				devs, err := s.store.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
					Targets: &core_apiv1.Targets{
						Ids: newOffline,
					},
					Status: &core_apiv1.UpdateDeviceRequest_Status{
						Online: toBoolRef(false),
					},
				})
				if err != nil {
					l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
					continue
				}
				l.Info().Str("route", "hearthbeat_pulsor").Str("event", "device_offline").Strs("new devices", newOffline).Msg("offline devices")
				s.hearthbeatsMutex.Lock()
				for _, d := range devs {
					err := s.m.Event().DeviceOffline().Publish("", d)
					if err != nil {
						l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device offline event")
					}
					delete(s.hearthbeats, d.Spec.DeviceId)
				}
				s.hearthbeatsMutex.Unlock()
			}

			l.Debug().Strs("new_offline_devices", newOffline).Msg("hearthbeats pulse")
		}
	}
}

func (s *CoreServer) hearthbeatSub(msg *mir.Msg, deviceId string) {
	l.Trace().Str("route", "hearthbeat").Msg("hearthbeat device request")
	// If not in map, mean is newly online device
	timeNow := time.Now().UTC()
	// if last != now by >= 3 mins, the device offline
	// every minute, a routine check the map to see if there is
	// hearthbeat older then 3 mins, if so set device to offline

	toBoolRef := func(b bool) *bool {
		return &b
	}
	// Since this update is only for hearthbeat and often, we dont want to have a device update event
	updReq := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{deviceId},
		},
		Status: &core_apiv1.UpdateDeviceRequest_Status{
			Online:         toBoolRef(true),
			LastHearthbeat: mir_models.AsProtoTimestamp(timeNow),
		},
	}
	dev, err := s.store.UpdateDevice(updReq)
	if err != nil {
		l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
		msg.Ack()
		return
	}
	// Means device is not in db, we provision it
	if len(dev) == 0 {
		_, err := s.m.Server().CreateDevice().Request(&core_apiv1.CreateDeviceRequest{
			Spec: &core_apiv1.Spec{
				DeviceId: deviceId,
			},
		})
		if err != nil {
			l.Error().Err(err).Str("deviceId", deviceId).Msg("could not automaticly provision new device")
			msg.Ack()
			return
		}
		dev, err = s.store.UpdateDevice(updReq)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
			msg.Ack()
			return
		}
	}

	if _, ok := s.hearthbeats[deviceId]; !ok && len(dev) > 0 {
		l.Info().Str("route", "hearthbeat").Str("event", "device_online").Msg(deviceId)
		err := s.m.Event().DeviceOnline().Publish(msg.GetOriginalTriggerId(), dev[0])
		if err != nil {
			l.Warn().Err(err).Str("device_id", deviceId).Msg("error occure while publishing device online event")
		}
	}
	s.hearthbeatsMutex.Lock()
	s.hearthbeats[deviceId] = time.Now().UTC()
	s.hearthbeatsMutex.Unlock()
	msg.Ack()
}

func sendReplyOrAck(bus *bus.BusConn, msg nats.Msg, m protoreflect.ProtoMessage) {
	if msg.Reply != "" {
		bResp, err := proto.Marshal(m)
		if err != nil {
			l.Error().Err(err).Msg("error occure while creating response")
		}
		err = bus.Publish(msg.Reply, bResp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while sending reply")
		}
	} else {
		msg.Ack()
	}
}
