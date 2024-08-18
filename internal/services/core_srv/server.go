package core_srv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type CoreServer struct {
	sub              *nats.Subscription
	bus              *bus.BusConn
	store            mng.DeviceStore
	hearthbeats      map[string]time.Time
	hearthbeatsMutex sync.RWMutex
}

var requestCount = metrics.NewCounterVec(prometheus.CounterOpts{
	Name: "request_count",
	Help: "Number of request for core",
}, []string{"route"})

var (
	l                   zerolog.Logger
	coreFunctionStreams = "*.*.core.v1alpha.*"
	offlineAfter        = time.Second * 30
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(requestCount)
}

func NewCore(logger zerolog.Logger, bus *bus.BusConn, store mng.DeviceStore) *CoreServer {
	l = logger.With().Str("srv", "core_server").Logger()
	hearbeats := map[string]time.Time{}

	// Subscribe to stream
	sub, err := bus.SubscribeSync(coreFunctionStreams)
	if err != nil {
		l.Error().Err(err).Msg("failed to subscribe to subject")
	}

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

	return &CoreServer{
		sub:         sub,
		bus:         bus,
		store:       store,
		hearthbeats: hearbeats,
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *CoreServer) Listen(ctx context.Context) {
	channelFns := map[string]chan nats.Msg{
		"create":     make(chan nats.Msg, 10),
		"delete":     make(chan nats.Msg, 10),
		"update":     make(chan nats.Msg, 10),
		"list":       make(chan nats.Msg, 10),
		"hearthbeat": make(chan nats.Msg, 10),
	}
	wg := &sync.WaitGroup{}

	go func() {
		wg.Add(1)
		s.createDeviceRequestHandler(channelFns["create"])
		wg.Done()
	}()
	go func() {
		wg.Add(1)
		go s.updateDeviceRequestHandler(channelFns["update"])
		wg.Done()
	}()
	go func() {
		wg.Add(1)
		go s.deleteDeviceRequestHandler(channelFns["delete"])
		wg.Done()
	}()
	go func() {
		wg.Add(1)
		go s.listDeviceRequestHandler(channelFns["list"])
		wg.Done()
	}()
	go func() {
		wg.Add(1)
		go s.hearthbeatRequestHandler(channelFns["hearthbeat"])
		wg.Done()
	}()
	go func() {
		wg.Add(1)
		go s.hearthbeatPulsor(ctx, time.Second*10, offlineAfter)
		wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			for _, v := range channelFns {
				close(v)
			}
			wg.Wait()
			l.Debug().Msg("core server shutdown")
			return
		default:
			msg, err := s.sub.NextMsgWithContext(ctx)
			if err != nil {
				if ctx.Err() != err {
					l.Error().Err(err).Msg("")
				}
				continue
			}

			route := clients.Subject(msg.Subject).GetFunction()
			l.Info().Str("route", route).Msg("core request")
			if _, exists := channelFns[route]; !exists {
				l.Warn().Str("route", route).Msg("route handler does not exist")
				continue
			}
			channelFns[route] <- *msg
			requestCount.WithLabelValues(route).Inc()
		}
	}
}

// TODO  add ability to create multiple devices by giving many ids
func (s *CoreServer) createDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		req := &core_apiv1.CreateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
				Response: &core_apiv1.CreateDeviceResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", req)).Msg("new device request")

		newDev, err := s.store.CreateDevice(req)
		if err != nil {
			if errors.Is(err, mir_models.ErrorInvalidDeviceID) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
					Response: &core_apiv1.CreateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    400,
							Message: err.Error(),
							Details: []string{"400 Bad Request"},
						},
					},
				})
			} else if errors.Is(err, mir_models.ErrorDeviceIdAlreadyExist) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
					Response: &core_apiv1.CreateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    409,
							Message: err.Error(),
							Details: []string{"409 Conflict"},
						},
					},
				})
			} else if errors.Is(err, mir_models.ErrorDbExecutingQuery) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
					Response: &core_apiv1.CreateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    500,
							Message: err.Error(),
							Details: []string{"500 Internal Server Error", err.Error()},
						},
					},
				})
			} else if errors.Is(err, mir_models.ErrorDbDeserializingResponse) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
					Response: &core_apiv1.CreateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    500,
							Message: err.Error(),
							Details: []string{"500 Internal Server Error", err.Error()},
						},
					},
				})
			}
			l.Error().Err(err).Msg("error occure while executing create device request")
			continue
		}

		// Publish created events
		for _, d := range newDev {
			err := core_client.PublishDeviceCreatedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("deviceId", d.Spec.DeviceId).Msg("error occure while publishing device created event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core_apiv1.CreateDeviceResponse{
			Response: &core_apiv1.CreateDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{
					Devices: mir_models.NewProtoDeviceListFromDevicesWithId(newDev),
				},
			}})
	}
}

func (s *CoreServer) updateDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		req := &core_apiv1.UpdateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core_apiv1.UpdateDeviceResponse{
				Response: &core_apiv1.UpdateDeviceResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "update").Str("payload", fmt.Sprintf("%v", req)).Msg("update device request")

		respDb, err := s.store.UpdateDevice(req)
		if err != nil {
			if errors.Is(err, mir_models.ErrorNoDeviceTargetProvided) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.UpdateDeviceResponse{
					Response: &core_apiv1.UpdateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    400,
							Message: err.Error(),
							Details: []string{"400 Bad Request"},
						},
					},
				})
			} else if errors.Is(err, mir_models.ErrorDbExecutingQuery) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.UpdateDeviceResponse{
					Response: &core_apiv1.UpdateDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    500,
							Message: err.Error(),
							Details: []string{"500 Internal Server Error", err.Error()},
						},
					},
				})
			}
			l.Error().Err(err).Msg("error occure while executing update device request")
			continue
		}

		// Publish update events
		for _, d := range respDb {
			err := core_client.PublishDeviceUpdatedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device updated event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core_apiv1.UpdateDeviceResponse{
			Response: &core_apiv1.UpdateDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{
					Devices: mir_models.NewProtoDeviceListFromDevicesWithId(respDb),
				},
			}})
	}
}

func (s *CoreServer) deleteDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		req := &core_apiv1.DeleteDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core_apiv1.DeleteDeviceResponse{
				Response: &core_apiv1.DeleteDeviceResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "delete").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

		devList, err := s.store.DeleteDevice(req)
		if err != nil {
			if errors.Is(err, mir_models.ErrorNoDeviceTargetProvided) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.DeleteDeviceResponse{
					Response: &core_apiv1.DeleteDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    400,
							Message: err.Error(),
							Details: []string{"400 Bad Request"},
						},
					},
				})
			} else if errors.Is(err, mir_models.ErrorDbExecutingQuery) {
				sendReplyOrAck(s.bus, msg, &core_apiv1.DeleteDeviceResponse{
					Response: &core_apiv1.DeleteDeviceResponse_Error{
						Error: &common_apiv1.Error{
							Code:    500,
							Message: err.Error(),
							Details: []string{"500 Internal Server Error", err.Error()},
						},
					},
				})
			}
			l.Error().Err(err).Msg("error occure while executing delete device request")
			continue
		}

		// Publish delete events
		for _, d := range devList {
			err := core_client.PublishDeviceDeletedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("deviceId", d.Spec.DeviceId).Msg("error occure while publishing device deleted event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core_apiv1.DeleteDeviceResponse{
			Response: &core_apiv1.DeleteDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{
					Devices: mir_models.NewProtoDeviceListFromDevicesWithId(devList),
				},
			}})
	}
}

func (s *CoreServer) listDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		req := &core_apiv1.ListDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core_apiv1.ListDeviceResponse{
				Response: &core_apiv1.ListDeviceResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "list").Str("payload", fmt.Sprintf("%v", req)).Msg("list device request")

		respDb, err := s.store.ListDevice(req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core_apiv1.ListDeviceResponse{
				Response: &core_apiv1.ListDeviceResponse_Error{
					Error: &common_apiv1.Error{
						Code:    500,
						Message: mir_models.ErrorDbExecutingQuery.Error(),
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		sendReplyOrAck(s.bus, msg, &core_apiv1.ListDeviceResponse{
			Response: &core_apiv1.ListDeviceResponse_Ok{
				Ok: &core_apiv1.DeviceList{
					Devices: mir_models.NewProtoDeviceListFromDevicesWithId(respDb),
				},
			}})
	}
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
			if len(newOffline) > 0 {
				toBoolRef := func(b bool) *bool {
					return &b
				}
				_, err := s.store.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
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
			}

			s.hearthbeatsMutex.RUnlock()
			s.hearthbeatsMutex.Lock()
			for _, key := range newOffline {
				l.Info().Str("route", "hearthbeat_pulsor").Str("event", "device_offline").Msg(key)
				err := core_client.PublishDeviceOfflineEvent(s.bus, key)
				if err != nil {
					l.Warn().Err(err).Str("device_id", key).Msg("error occure while publishing device offline event")
				}
				delete(s.hearthbeats, key)
			}
			s.hearthbeatsMutex.Unlock()
			l.Debug().Strs("new_offline_devices", newOffline).Msg("hearthbeats pulse")
		}
	}
}

func (s *CoreServer) hearthbeatRequestHandler(ch chan nats.Msg) {
	for {
		msg, ok := <-ch
		if !ok {
			return
		}
		l.Debug().Str("route", "hearthbeat").Msg("hearthbeat device request")
		deviceId := clients.Subject(msg.Subject).GetId()
		// If not in map, mean is newly online device
		s.hearthbeatsMutex.Lock()
		if _, ok := s.hearthbeats[deviceId]; !ok {
			l.Info().Str("route", "hearthbeat").Str("event", "device_online").Msg(deviceId)
			err := core_client.PublishDeviceOnlineEvent(s.bus, deviceId)
			if err != nil {
				l.Warn().Err(err).Str("device_id", deviceId).Msg("error occure while publishing device online event")
			}
		}
		s.hearthbeats[deviceId] = time.Now().UTC()
		s.hearthbeatsMutex.Unlock()
		// map[deviceid]lasthearthbeat
		// if last != now by >= 3 mins, the device offline
		// every minute, a routine check the map to see if there is
		// hearthbeat older then 3 mins, if so set device to offline

		toBoolRef := func(b bool) *bool {
			return &b
		}
		_, err := s.store.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{deviceId},
			},
			Status: &core_apiv1.UpdateDeviceRequest_Status{
				Online:         toBoolRef(true),
				LastHearthbeat: mir_models.AsProtoTimestamp(s.hearthbeats[deviceId]),
			},
		})
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
			msg.Ack()
			continue
		}
		msg.Ack()
	}
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
