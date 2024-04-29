package core

import (
	"context"
	"fmt"
	"strings"

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
	sub *nats.Subscription
	bus *bus.BusConn
	db  *surrealdb.DB
}
type Device struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
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
	return &CoreServer{
		sub: sub,
		bus: bus,
		db:  db,
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *CoreServer) Listen(ctx context.Context) {
	channelFns := map[string]chan nats.Msg{
		"create": make(chan nats.Msg, 10),
		"delete": make(chan nats.Msg, 10),
		"update": make(chan nats.Msg, 10),
		"list":   make(chan nats.Msg, 10),
	}
	go s.createDeviceRequestHandler(channelFns["create"])
	go s.updateDeviceRequestHandler(channelFns["update"])
	go s.deleteDeviceRequestHandler(channelFns["delete"])
	go s.listDeviceRequestHandler(channelFns["list"])

	select {
	case <-ctx.Done():
		l.Info().Msg("shutting down")
		return
	default:
		for {
			msg, err := s.sub.NextMsgWithContext(ctx)
			if err != nil {
				l.Error().Err(err).Msg("")
				continue
			}
			route := getRoutingFunc(msg.Subject)
			l.Info().Str("route", route).Msg("device request")

			channelFns[route] <- *msg
			requestCount.WithLabelValues(route).Inc()
		}
	}
}

// TODO check if unique on device_id
func (s *CoreServer) createDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
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

		q, v := createListQueryForDevice(&core.ListDeviceRequest{
			Targets: &core.Targets{
				Ids: []string{req.DeviceId},
			},
		})
		respCheck, err := executeQueryForType[[]*core.Device](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
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

		respDb, err := s.db.Create("devices", req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while executing a db query")
			sendReplyOrAck(s.bus, msg, &core.CreateDeviceResponse{
				Response: &core.CreateDeviceResponse_Error{
					Error: &core.Error{
						Code:    500,
						Message: "error occure while executing a db query",
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
			continue
		}
		newDev := make([]core.CreateDeviceRequest, 1)
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
				Ok: &core.DeviceIdList{
					DeviceIds: []string{newDev[0].DeviceId},
				},
			}})
	}
}

func (s *CoreServer) updateDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
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
		q, v := createUpdateQueryForDevice(req)
		respDb, err := executeQueryForType[[]core.Device](s.db, q, v)
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

		ids := []string{}
		for _, v := range respDb {
			ids = append(ids, v.DeviceId)
		}

		sendReplyOrAck(s.bus, msg, &core.UpdateDeviceResponse{
			Response: &core.UpdateDeviceResponse_Ok{
				Ok: &core.DeviceIdList{
					DeviceIds: ids,
				},
			}})
	}
}

func (s *CoreServer) deleteDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
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

		q, v := createDeleteQueryForDevice(req)
		respDb, err := executeQueryForType[[]core.Device](s.db, q, v)
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

		ids := []string{}
		for _, v := range respDb {
			ids = append(ids, v.DeviceId)
		}

		sendReplyOrAck(s.bus, msg, &core.DeleteDeviceResponse{
			Response: &core.DeleteDeviceResponse_Ok{
				Ok: &core.DeviceIdList{
					DeviceIds: ids,
				},
			}})
	}
}

func (s *CoreServer) listDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
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
		l.Debug().Str("route", "list").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

		q, v := createListQueryForDevice(req)
		respDb, err := executeQueryForType[[]*core.Device](s.db, q, v)
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
					Devices: respDb,
				},
			}})
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

func createUpdateQueryForDevice(req *core.UpdateDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}
	q.WriteString("UPDATE devices MERGE {")
	if req.Description != nil {
		q.WriteString("description: $DESC,")
		vars["DESC"] = *req.Description
	}
	if req.Labels != nil && len(req.Labels) > 0 {
		q.WriteString("labels: {")
		for key, val := range req.Labels {
			q.WriteString(key)
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
	if req.Annotations != nil && len(req.Annotations) > 0 {
		q.WriteString("annotations: {")
		for key, val := range req.Annotations {
			q.WriteString(key)
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

	q.WriteString("} WHERE ")

	cond := []string{}
	if len(req.Targets.Ids) > 0 {
		var t []string
		for _, id := range req.Targets.Ids {
			t = append(t, fmt.Sprintf("device_id = \"%s\"", id))
		}
		cond = append(cond, strings.Join(t, " OR "))
	}
	if len(req.Targets.Labels) > 0 {
		var t []string
		for k, v := range req.Targets.Labels {
			t = append(t, fmt.Sprintf("labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " OR "))
	q.WriteString(";")
	sql = q.String()

	return
}

func createDeleteQueryForDevice(req *core.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.Targets.Ids, req.Targets.Labels, req.Targets.Annotations))

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

func createWhereStatementWithTargets(targetIds []string, targetLabels map[string]string, targetAnno map[string]string) string {
	var q strings.Builder

	cond := []string{}
	if len(targetIds) > 0 {
		var t []string
		for _, id := range targetIds {
			t = append(t, fmt.Sprintf("device_id = \"%s\"", id))
		}
		cond = append(cond, strings.Join(t, " OR "))
	}
	if len(targetLabels) > 0 {
		var t []string
		for k, v := range targetLabels {
			t = append(t, fmt.Sprintf("labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	if len(targetAnno) > 0 {
		var t []string
		for k, v := range targetAnno {
			t = append(t, fmt.Sprintf("annotations.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " OR "))
	q.WriteString(";")
	return q.String()
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
