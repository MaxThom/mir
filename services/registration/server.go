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
	}
	go s.createDeviceRequestHandler(channelFns["create"])

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
			channelFns[route] <- *msg
			requestCount.WithLabelValues(route).Inc()
		}
	}
}

func (s *RegistrationServer) createDeviceRequestHandler(ch chan nats.Msg) {
	for {
		msg := <-ch
		req := &registration.CreateDeviceRequest{}
		err := proto.Unmarshal(msg.Data, req)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}
		l.Info().Str("route", "create").Str("payload", fmt.Sprintf("%v", req)).Msg("new device request")

		if _, err = s.db.Use("global", "mir"); err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}

		// TODO return msg with reply with device id
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
			Msg:      "Device created",
		}
		bResp, err := proto.Marshal(resp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
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

func getRoutingFunc(s string) string {
	index := strings.LastIndex(s, ".")
	if index == -1 {
		return ""
	}
	return s[index+1:]
}
