package registration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	"github.com/maxthom/mir/libs/api/metrics"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
)

type RegistrationServer struct {
	cons jetstream.Consumer
	bus  *bus.BusConn
	db   *surrealdb.DB
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

func NewRegistrationServer(logger zerolog.Logger, bus *bus.BusConn, cons jetstream.Consumer, db *surrealdb.DB) *RegistrationServer {
	l = logger.With().Str("srv", "registration_server").Logger()
	return &RegistrationServer{
		cons: cons,
		bus:  bus,
		db:   db,
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *RegistrationServer) Listen(ctx context.Context) {
	channelFns := map[string]chan jetstream.Msg{
		"create": make(chan jetstream.Msg, 10),
		"delete": make(chan jetstream.Msg, 10),
		"update": make(chan jetstream.Msg, 10),
	}
	go s.createDeviceRequestHandler(channelFns["create"])

	select {
	case <-ctx.Done():
		l.Info().Msg("shutting down")
		return
	default:
		for {
			msgs, err := s.cons.Fetch(10, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				l.Error().Err(err).Msg("")
			}

			for msg := range msgs.Messages() {
				route := getRoutingFunc(msg.Subject())
				channelFns[route] <- msg
				requestCount.WithLabelValues(route).Inc()
			}
			if msgs.Error() != nil {
				l.Error().Err(err).Msg("error while fetching from bus")
			}
		}
	}
}

func (s *RegistrationServer) createDeviceRequestHandler(ch chan jetstream.Msg) {
	for {
		msg := <-ch
		req := &registration.CreateDeviceRequest{}
		err := proto.Unmarshal(msg.Data(), req)
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
		fmt.Println(resp)
		fmt.Println("CACA")
		fmt.Println(respDb)
		bResp, err := proto.Marshal(resp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
		}

		err = s.bus.Publish(msg.Reply(), bResp)
		if err != nil {
			l.Error().Err(err).Msg("error occure while using db")
			continue
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
