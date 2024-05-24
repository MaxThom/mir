package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
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
	sub         *nats.Subscription
	bus         *bus.BusConn
	db          *surrealdb.DB
	hearthbeats map[string]time.Time
}

var requestCount = metrics.NewCounterVec(prometheus.CounterOpts{
	Name: "request_count",
	Help: "Number of request for core",
}, []string{"route"})

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(requestCount)
}

func NewCore(logger zerolog.Logger, bus *bus.BusConn, sub *nats.Subscription, db *surrealdb.DB) *CoreServer {
	l = logger.With().Str("srv", "core_server").Logger()
	hearbeats := map[string]time.Time{}

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
			hearbeats[d.Meta.DeviceId] = d.Status.LastHearthbeat
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
		go s.hearthbeatPulsor(ctx, time.Second*10, time.Second*30)
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
			route := getRoutingFunc(msg.Subject)
			l.Info().Str("route", route).Msg("device request")

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
		newDev := []*DeviceWithId{}
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
		if spec, ok := req.Request.(*core.UpdateDeviceRequest_Meta_); ok && spec != nil {
			q, v = createUpdateQueryForDeviceMeta(req.Targets, spec.Meta)
		} else if status, ok := req.Request.(*core.UpdateDeviceRequest_Status_); ok && status != nil {
			// TODO status update
		}

		respDb, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
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

		// TODO find a way for delete to return the device documents
		q, v := createDeleteQueryForDevice(req)
		respDb, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
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

		sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
			Response: &core.DeleteDeviceResponse_Ok{
				Ok: &core.DeviceList{
					Devices: NewProtoDeviceListFromDevicesWithId(respDb),
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
		respDb, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
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
			newOffline := []string{}
			now := time.Now().UTC()
			for k, v := range s.hearthbeats {
				if v.Add(offlineAfter).Before(now) {
					newOffline = append(newOffline, k)
					// TODO create offline event
					q, v := createHeartbeatQuery(k, time.Time{}, false)
					_, err := executeQueryForType[[]*DeviceWithId](s.db, q, v)
					if err != nil {
						l.Error().Err(err).Msg("error occure while executing hearthbeat db query")
						continue
					}
				}
			}
			// TODO some lock on that map
			for _, key := range newOffline {
				delete(s.hearthbeats, key)
			}
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
		deviceId := getDeviceIdFromSubject(msg.Subject)
		s.hearthbeats[deviceId] = time.Now().UTC()
		// TODO compute using a map
		// map[deviceid]lasthearthbeat
		// if last != now by >= 3 mins, the device offline
		// every minute, a routine check the map to see if there is
		// hearthbeat older then 3 mins, if so set device to offline
		// TODO tui terminal, 10secs refresh list silently
		// or maybe r hotkey?
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

func getRoutingFunc(s string) string {
	index := strings.LastIndex(s, ".")
	if index == -1 {
		return ""
	}
	return s[index+1:]
}

func createUpdateQueryForDeviceMeta(t *core.Targets, spec *core.UpdateDeviceRequest_Meta) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	q.WriteString("meta: {")
	if spec.Name != nil {
		q.WriteString("name: $NAME,")
		vars["NAME"] = *spec.Name
	}
	if spec.Disabled != nil {
		q.WriteString("disabled: $DISA,")
		vars["DISA"] = *spec.Disabled
	}
	if spec.Labels != nil && len(spec.Labels) > 0 {
		q.WriteString("labels: {")
		for key, val := range spec.Labels {
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
	if spec.Annotations != nil && len(spec.Annotations) > 0 {
		q.WriteString("annotations: {")
		for key, val := range spec.Annotations {
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

	q.WriteString("},} WHERE ")
	q.WriteString(createWhereStatementWithTargets(t.Ids, t.Labels, t.Annotations))
	q.WriteString(";")
	sql = q.String()

	return
}

func createDeleteQueryForDevice(req *core.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.Targets.Ids, req.Targets.Labels, req.Targets.Annotations))
	q.WriteString(";")
	sql = q.String()
	return
}

func createListQueryForDevice(req *core.ListDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM devices")
	where := createWhereStatementWithTargets(req.Targets.Ids, req.Targets.Labels, req.Targets.Annotations)
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
	q.WriteString(createWhereStatementWithTargets([]string{id}, nil, nil))
	q.WriteString(";")
	sql = q.String()
	return
}

// TODO find how we can query using / in name in WHERE clause
func createWhereStatementWithTargets(targetIds []string, targetLabels map[string]string, targetAnno map[string]string) string {
	var q strings.Builder

	cond := []string{}
	if len(targetIds) > 0 {
		var t []string
		for _, id := range targetIds {
			t = append(t, fmt.Sprintf("meta.deviceId = \"%s\"", id))
		}
		cond = append(cond, strings.Join(t, " OR "))
	}
	if len(targetLabels) > 0 {
		var t []string
		for k, v := range targetLabels {
			t = append(t, fmt.Sprintf("meta.labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	if len(targetAnno) > 0 {
		var t []string
		for k, v := range targetAnno {
			t = append(t, fmt.Sprintf("meta.annotations.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " OR "))
	ti := q.String()
	return ti
}

func getDeviceIdFromSubject(s string) string {
	pos := strings.Index(s, ".")
	if pos == -1 {
		return ""
	}
	return s[:pos]
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
