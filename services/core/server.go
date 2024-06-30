package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/api/routes"
	"github.com/maxthom/mir/libs/api/metrics"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type CoreServer struct {
	sub              *nats.Subscription
	bus              *bus.BusConn
	db               *surrealdb.DB
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

func NewCore(logger zerolog.Logger, bus *bus.BusConn, db *surrealdb.DB) *CoreServer {
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
	q, v := createListQueryForDevice(&core.ListDeviceRequest{Targets: &core.Targets{}})
	devices, err := executeQueryForType[[]*DeviceWithId](db, q, v)
	if err != nil {
		l.Error().Err(err).Msg("error occure while executing list query")
	}
	for _, d := range devices {
		// We only add the online ones
		// This way, the pulse doesnt do a check on offline device
		// When an offline device sends a first pulse, it get added
		// to the map.
		// If a device becomes offline, it's removed from the map
		if d != nil && d.Status.Online {
			hearbeats[d.Spec.DeviceId] = d.Status.LastHearthbeat
		}
	}

	return &CoreServer{
		sub:         sub,
		bus:         bus,
		db:          db,
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

			route := routes.Subject(msg.Subject).GetFunction()
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
		req := &core.CreateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "error occure while unmarhsalling request payload",
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", req)).Msg("new device request")

		if req.DeviceId == "" {
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "Invalid device ID",
						Details: []string{"400 Bad Request"},
					},
				},
			})
			continue
		}

		q, v := createListQueryForDevice(&core.ListDeviceRequest{
			Targets: &core.Targets{
				Ids: []string{req.DeviceId},
			},
		})
		respCheck, err := executeQueryForType[[]*core.Device](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing list db query")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing list db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		if len(respCheck) > 0 {
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    409,
						Message: "a device with the same id already exists",
						Details: []string{"409 Conflict"},
					},
				},
			})
			continue
		}

		respDb, err := s.db.Create("devices", NewDeviceFromCreateDeviceReq(req))
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing create db query")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing create db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}
		newDev := []DeviceWithId{}
		err = surrealdb.Unmarshal(respDb, &newDev)
		if err != nil {
			l.Error().Err(err).Msg("error occure while deserializing a db response")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while deserializing a db response",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		// Publish created events
		for _, d := range newDev {
			err := PublishDeviceCreatedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device created event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
			Response: &core.CreateDeviceResponse_Ok{
				Ok: &core.DeviceList{
					Devices: NewProtoDeviceListFromDevicesWithId(newDev),
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
		req := &core.UpdateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core.UpdateDeviceResponse{
				Response: &core.UpdateDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "error occure while unmarhsalling request payload",
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "update").Str("payload", fmt.Sprintf("%v", req)).Msg("update device request")

		if req.Targets == nil ||
			len(req.Targets.Ids) == 0 &&
				len(req.Targets.Names) == 0 &&
				len(req.Targets.Namespaces) == 0 &&
				len(req.Targets.Labels) == 0 &&
				len(req.Targets.Annotations) == 0 {
			sendReplyOrAck(s.bus, msg, &core.UpdateDeviceResponse{
				Response: &core.UpdateDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "no target provided for update",
						Details: []string{"400 Bad Request"},
					},
				},
			})
			continue
		}

		// Update is full document
		// Change is a merge
		// Modify is a patch

		// TODO check how to do it on type
		q := ""
		v := map[string]any{}
		q, v = createUpdateQueryForDevice(req.Targets, req)
		respDb, err := executeQueryForType[[]DeviceWithId](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.UpdateDeviceResponse{
				Response: &core.UpdateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		// Publish update events
		for _, d := range respDb {
			err := PublishDeviceUpdatedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device updated event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core.UpdateDeviceResponse{
			Response: &core.UpdateDeviceResponse_Ok{
				Ok: &core.DeviceList{
					Devices: NewProtoDeviceListFromDevicesWithId(respDb),
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
		req := &core.DeleteDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
				Response: &core.DeleteDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "error occure while unmarhsalling request payload",
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "delete").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

		if req.Targets == nil ||
			len(req.Targets.Ids) == 0 &&
				len(req.Targets.Names) == 0 &&
				len(req.Targets.Namespaces) == 0 &&
				len(req.Targets.Labels) == 0 &&
				len(req.Targets.Annotations) == 0 {
			sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
				Response: &core.DeleteDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "no target provided for delete",
						Details: []string{"400 Bad Request"},
					},
				},
			})
			continue
		}

		qList, vList := createListQueryForDevice(&core.ListDeviceRequest{
			Targets: req.Targets,
		})
		respDbList, err := executeQueryForType[[]DeviceWithId](s.db, qList, vList)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
				Response: &core.DeleteDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		// IDEA would be nice that surreal delete statement returns the document
		// so we dont have to do two queries
		q, v := createDeleteQueryForDevice(req)
		_, err = executeQueryForType[[]DeviceWithId](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
				Response: &core.DeleteDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		// Publish delete events
		for _, d := range respDbList {
			err := PublishDeviceDeletedEvent(s.bus, d.Spec.DeviceId, d)
			if err != nil {
				l.Warn().Err(err).Str("device_id", d.Spec.DeviceId).Msg("error occure while publishing device deleted event")
			}
		}

		sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
			Response: &core.DeleteDeviceResponse_Ok{
				Ok: &core.DeviceList{
					Devices: NewProtoDeviceListFromDevicesWithId(respDbList),
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
		req := &core.ListDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling request payload")
			sendReplyOrAck(s.bus, msg, &core.ListDeviceResponse{
				Response: &core.ListDeviceResponse_Error{
					Error: &core.Error{
						Code:    400,
						Message: "error occure while unmarhsalling request payload",
						Details: []string{"400 Bad Request", err.Error()},
					},
				},
			})
			continue
		}
		l.Debug().Str("route", "list").Str("payload", fmt.Sprintf("%v", req)).Msg("list device request")

		q, v := createListQueryForDevice(req)
		respDb, err := executeQueryForType[[]DeviceWithId](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.ListDeviceResponse{
				Response: &core.ListDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}

		sendReplyOrAck(s.bus, msg, &core.ListDeviceResponse{
			Response: &core.ListDeviceResponse_Ok{
				Ok: &core.DeviceList{
					Devices: NewProtoDeviceListFromDevicesWithId(respDb),
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
					// TODO could be one query that does all the now offline devices instead of many
					q, v := createHeartbeatQuery(k, time.Time{}, false)
					_, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
					if err != nil {
						l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
						continue
					}
				}
			}
			s.hearthbeatsMutex.RUnlock()
			s.hearthbeatsMutex.Lock()
			for _, key := range newOffline {
				l.Info().Str("route", "hearthbeat_pulsor").Str("event", "device_offline").Msg(key)
				err := PublishDeviceOfflineEvent(s.bus, key)
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
		deviceId := routes.Subject(msg.Subject).GetId()
		// If not in map, mean is newly online device
		s.hearthbeatsMutex.Lock()
		if _, ok := s.hearthbeats[deviceId]; !ok {
			l.Info().Str("route", "hearthbeat").Str("event", "device_online").Msg(deviceId)
			err := PublishDeviceOnlineEvent(s.bus, deviceId)
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
		q, v := createHeartbeatQuery(deviceId, s.hearthbeats[deviceId], true)
		_, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
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

func createUpdateQueryForDevice(t *core.Targets, upd *core.UpdateDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	if upd.Meta != nil {
		q.WriteString("meta: {")
		if upd.Meta.Name != nil {
			q.WriteString("name: $NAME,")
			vars["NAME"] = *upd.Meta.Name
		}
		if upd.Meta.Namespace != nil {
			q.WriteString("namespace: $NS,")
			vars["NS"] = *upd.Meta.Namespace
		}
		if upd.Meta.Labels != nil && len(upd.Meta.Labels) > 0 {
			q.WriteString("labels: {")
			for key, val := range upd.Meta.Labels {
				q.WriteString("\"")
				q.WriteString(key)
				q.WriteString("\"")
				q.WriteString(": ")
				if val == nil || val.Value == nil {
					q.WriteString("NONE")
				} else {
					q.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
				}
				q.WriteString(",")
			}
			q.WriteString("},")
		}
		if upd.Meta.Annotations != nil && len(upd.Meta.Annotations) > 0 {
			q.WriteString("annotations: {")
			for key, val := range upd.Meta.Annotations {
				q.WriteString("\"")
				q.WriteString(key)
				q.WriteString("\"")
				q.WriteString(": ")
				if val == nil || val.Value == nil {
					q.WriteString("NONE")
				} else {
					q.WriteString(fmt.Sprintf("\"%s\"", val.GetValue()))
				}
				q.WriteString(",")
			}
			q.WriteString("},")
		}
		q.WriteString("},")
	}
	if upd.Spec != nil {
		q.WriteString("spec: {")
		if upd.Spec.Disabled != nil {
			q.WriteString("disabled: $DIS,")
			vars["DIS"] = *upd.Spec.Disabled
		}
		q.WriteString("},")
	}
	if upd.Status != nil {
		q.WriteString("status: {")
		if upd.Status.LastHearthbeat != nil && !AsGoTime(upd.Status.LastHearthbeat).IsZero() {
			q.WriteString("lastHearthbeat: $BEAT,")
			vars["BEAT"] = t
		}
		if upd.Status.Online != nil {
			q.WriteString("online: $ON,")
			vars["ON"] = upd.Status.Online
		}
		q.WriteString("},")
	}

	q.WriteString("} WHERE ")
	q.WriteString(createWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()

	return
}

func createUpdateQueryForDeviceSpec(t *core.Targets, upd *core.UpdateDeviceRequest_Spec) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	q.WriteString("spec: {")
	if upd.Disabled != nil {
		q.WriteString("disabled: $DIS,")
		vars["DIS"] = *upd.Disabled
	}

	q.WriteString("},} WHERE ")
	q.WriteString(createWhereStatementWithTargets(t))
	q.WriteString(";")
	sql = q.String()

	return
}

func createDeleteQueryForDevice(req *core.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.Targets))
	q.WriteString(";")
	sql = q.String()
	return
}

func createListQueryForDevice(req *core.ListDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM devices")
	where := createWhereStatementWithTargets(req.Targets)
	if len(where) > 0 {
		q.WriteString(" WHERE ")
		q.WriteString(where)
	}

	q.WriteString(";")
	sql = q.String()
	return
}

func createHeartbeatQuery(id string, t time.Time, online bool) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	q.WriteString("status: {")
	if !t.IsZero() {
		q.WriteString("lastHearthbeat: $BEAT,")
		vars["BEAT"] = t
	}
	q.WriteString("online: $ON,")
	vars["ON"] = online
	q.WriteString("},} WHERE ")
	q.WriteString(createWhereStatementWithTargets(&core.Targets{
		Ids: []string{id},
	}))
	q.WriteString(";")
	sql = q.String()
	return
}

// TODO find how we can query using / in name in WHERE clause
func createWhereStatementWithTargets(t *core.Targets) string {
	var q strings.Builder
	if t == nil {
		return ""
	}

	cond := []string{}
	if len(t.Ids) > 0 {
		var i []string
		for _, id := range t.Ids {
			i = append(i, fmt.Sprintf("spec.deviceId = \"%s\"", id))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Names) > 0 {
		var i []string
		for _, ns := range t.Names {
			i = append(i, fmt.Sprintf("meta.name = \"%s\"", ns))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Namespaces) > 0 {
		var i []string
		for _, ns := range t.Namespaces {
			i = append(i, fmt.Sprintf("meta.namespace = \"%s\"", ns))
		}
		cond = append(cond, strings.Join(i, " OR "))
	}
	if len(t.Labels) > 0 {
		var i []string
		for k, v := range t.Labels {
			i = append(i, fmt.Sprintf("meta.labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	if len(t.Annotations) > 0 {
		var i []string
		for k, v := range t.Annotations {
			i = append(i, fmt.Sprintf("meta.annotations.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	// TODO switch this to AND, must add ( ) above
	q.WriteString(strings.Join(cond, " OR "))
	ti := q.String()
	return ti
}

func executeQueryForType[T any](db *surrealdb.DB, query string, vars map[string]any) (T, error) {
	var empty T
	result, err := db.Query(query, vars)
	if err != nil {
		return empty, err
	}

	res, err := surrealdb.SmartUnmarshal[T](result, err)
	if err != nil {
		return empty, err
	}

	return res, nil
}
