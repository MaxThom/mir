package registration

import (
	"context"
	"fmt"
	"strings"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	"github.com/maxthom/mir/libs/api/metrics"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type RegistrationServer struct {
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
	Help: "Number of request for registration",
}, []string{"route"})

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(requestCount)
}

func NewRegistrationServer(logger zerolog.Logger, bus *bus.BusConn, sub *nats.Subscription, db *surrealdb.DB) *RegistrationServer {
	l = logger.With().Str("srv", "registration_server").Logger()
	return &RegistrationServer{
		sub: sub,
		bus: bus,
		db:  db,
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *RegistrationServer) Listen(ctx context.Context) {
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
			}
			route := getRoutingFunc(msg.Subject)
			l.Info().Str("route", route).Msg("device request")

			channelFns[route] <- *msg
			requestCount.WithLabelValues(route).Inc()
		}
	}
}

// TODO check if unique on device_id
func (s *RegistrationServer) createDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
		req := &registration.CreateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarhsalling payload")
			continue
		}
		l.Debug().Str("route", "create").Str("payload", fmt.Sprintf("%v", req)).Msg("new device request")

		if _, err = s.db.Use("global", "mir"); err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}

		respDb, err := s.db.Create("devices", req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}
		newDev := make([]registration.CreateDeviceRequest, 1)
		err = surrealdb.Unmarshal(respDb, &newDev)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}

		resp := &registration.CreateDeviceResponse{
			DeviceId: newDev[0].DeviceId,
			Msg:      []string{"Device created"},
		}
		bResp, err := proto.Marshal(resp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while creating response")
			continue
		}

		if msg.Reply != "" {
			err = s.bus.Publish(msg.Reply, bResp)
			if err != nil {
				l.Error().Err(err).Msg("error occure while sending reply")
				continue
			}
		}

		msg.Ack()
	}
}

func (s *RegistrationServer) updateDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
		resp := &registration.UpdateDeviceResponse{}

		req := &registration.UpdateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarshalling request")
			resp.Msg = append(resp.Msg, "error occure while unmarshalling request")
			sendReplyIfRequest(s.bus, msg.Reply, resp)
			continue
		}
		l.Debug().Str("route", "update").Str("payload", fmt.Sprintf("%v", req)).Msg("update device request")

		if _, err = s.db.Use("global", "mir"); err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			resp.Msg = append(resp.Msg, "error occure while unmarshalling request")
			sendReplyIfRequest(s.bus, msg.Reply, resp)
			continue
		}

		// Update is full document
		// Change is a merge
		// Modify is a patch
		q, v := createUpdateQueryForDevice(req)
		respDb, err := executeQueryForType[[]registration.Device](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while updating records")
			resp.Msg = append(resp.Msg, "error occure while updating records")
			continue
		}

		for _, v := range respDb {
			resp.AffectedDevices = append(resp.AffectedDevices, v.DeviceId)
		}

		sendReplyIfRequest(s.bus, msg.Reply, resp)

		msg.Ack()
	}
}

func (s *RegistrationServer) deleteDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
		req := &registration.DeleteDeviceRequest{}
		resp := &registration.DeleteDeviceResponse{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarshalling delete request")
			resp.Msg = append(resp.Msg, "error occure while unmarshalling delete request")
			continue
		}
		l.Debug().Str("route", "delete").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

		if _, err = s.db.Use("global", "mir"); err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			resp.Msg = append(resp.Msg, "error occure while using db")
			continue
		}

		q, v := createDeleteQueryForDevice(req)
		respDb, err := executeQueryForType[[]registration.Device](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			resp.Msg = append(resp.Msg, "error occure while using db")
			continue
		}

		for _, v := range respDb {
			resp.AffectedDevices = append(resp.AffectedDevices, v.DeviceId)
		}

		sendReplyIfRequest(s.bus, msg.Reply, resp)

		msg.Ack()
	}
}

func (s *RegistrationServer) listDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
		req := &registration.ListDeviceRequest{}
		resp := &registration.ListDeviceResponse{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while unmarshalling delete request")
			resp.Msg = append(resp.Msg, "error occure while unmarshalling delete request")
			continue
		}
		l.Info().Str("route", "list").Str("payload", fmt.Sprintf("%v", req)).Msg("delete device request")

		if _, err = s.db.Use("global", "mir"); err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			resp.Msg = append(resp.Msg, "error occure while using db")
			continue
		}

		q, v := createListQueryForDevice(req)
		respDb, err := executeQueryForType[[]*registration.Device](s.db, q, v)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			resp.Msg = append(resp.Msg, "error occure while using db")
			continue
		}
		resp.Devices = respDb

		sendReplyIfRequest(s.bus, msg.Reply, resp)

		msg.Ack()
	}
}

func sendReplyIfRequest(bus *bus.BusConn, replyId string, m protoreflect.ProtoMessage) {
	if replyId != "" {
		bResp, err := proto.Marshal(m)
		if err != nil {
			l.Error().Err(err).Msg("error occure while creating response")
		}
		err = bus.Publish(replyId, bResp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while sending reply")
		}
	}
}

func getRoutingFunc(s string) string {
	index := strings.LastIndex(s, ".")
	if index == -1 {
		return ""
	}
	return s[index+1:]
}

func createUpdateQueryForDevice(req *registration.UpdateDeviceRequest) (sql string, vars map[string]any) {
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
	if len(req.TargetIds) > 0 {
		var t []string
		for _, id := range req.TargetIds {
			t = append(t, fmt.Sprintf("device_id = \"%s\"", id))
		}
		cond = append(cond, strings.Join(t, " OR "))
	}
	if len(req.TargetLabels) > 0 {
		var t []string
		for k, v := range req.TargetLabels {
			t = append(t, fmt.Sprintf("labels.%s = \"%s\"", k, v))
		}
		cond = append(cond, "("+strings.Join(t, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " OR "))
	q.WriteString(";")
	sql = q.String()

	return
}

func createDeleteQueryForDevice(req *registration.DeleteDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("DELETE FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.TargetIds, req.TargetLabels))

	sql = q.String()
	return
}

func createListQueryForDevice(req *registration.ListDeviceRequest) (sql string, vars map[string]any) {
	var q strings.Builder
	vars = map[string]any{}

	q.WriteString("SELECT * FROM devices WHERE ")
	q.WriteString(createWhereStatementWithTargets(req.TargetIds, req.TargetLabels))

	sql = q.String()
	return
}

func createWhereStatementWithTargets(targetIds []string, targetLabels map[string]string) string {
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
