package registration

import (
	"context"
	"fmt"
	"time"

	"github.com/maxthom/mir/libs/api/metrics"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
)

type RegistrationServer struct {
	cons jetstream.Consumer
	db   *surrealdb.DB
}
type Device struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

var (
	uploadMetric = metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	datapointCount = metrics.NewCounter(prometheus.CounterOpts{
		Name: "datapoint_count",
		Help: "Number of datapoint fed into protoproxy from nats",
	})
)

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(uploadMetric)
	reg.Register(datapointCount)
}

func NewRegistrationServer(logger zerolog.Logger, cons jetstream.Consumer, db *surrealdb.DB) *RegistrationServer {
	l = logger.With().Str("srv", "registration_server").Logger()
	return &RegistrationServer{
		cons: cons,
		db:   db,
	}
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *RegistrationServer) Listen(ctx context.Context) {

	// Create routine for deserializing and writing to db
	// dataCh := make(chan ProtoPayload)
	// go func(stream <-chan ProtoPayload) {
	// 	for pr := range stream {
	// 		lp, err := p.marshallers.Deserialize(pr.data, pr.key)
	// 		if err != nil {
	// 			l.Error().Err(err).Msg("error while marshalling")
	// 			continue
	// 		}
	// 		// fmt.Println(lp)
	// 		// This is also another channel and routine to write to db
	// 		p.writer.WriteRecord(lp)
	// 	}
	// }(dataCh)

	l.Info().Msg("listening to devices messages")
	select {
	case <-ctx.Done():
		l.Info().Msg("shutting down")
		return
	default:
		for {
			msgs, err := s.cons.Fetch(100, jetstream.FetchMaxWait(1*time.Second))
			if err != nil {
				l.Error().Err(err).Msg("")
			}

			for msg := range msgs.Messages() {
				// msgName, ok := msg.Headers()["__pb"]
				// if !ok {
				// 	l.Warn().Err(err).Msg("missing proto header")
				// 	continue
				// }
				// deviceId := ""
				// deviceIdAr, ok := msg.Headers()["deviceId"]
				// if ok {
				// 	deviceId = deviceIdAr[0]
				// }

				// dataCh <- ProtoPayload{
				// 	data: msg.Data(),
				// 	key: MarshallerKey{
				// 		messageName: msgName[0],
				// 		deviceId:    deviceId,
				// 	},
				// }
				//
				fmt.Println(string(msg.Data()))

				// TODO business logic.
				// Filter on which function (create, delete, update, heartbeat)
				// Could be another consumer for heathbeat
				// Thinking of not using channel, else the Ack of message
				// will not be true if the software crashes
				// or if we can ack with the message as payload
				// With one channel per function, and msg as payoad

				msg.Ack()
				// TODO Adjust to reduce observability cost
				datapointCount.Inc()
			}
			if msgs.Error() != nil {
				l.Error().Err(err).Msg("error while fetching from bus")
			}
		}
	}

	var err error
	if _, err = s.db.Use("test", "test"); err != nil {
		panic(err)
	}

	dev := Device{
		Name: "edge1",
	}

	data, err := s.db.Create("devices", dev)
	if err != nil {
		panic(err)
	}
	fmt.Println(data)

	// Unmarshal data
	createdDev := make([]Device, 1)
	err = surrealdb.Unmarshal(data, &createdDev)
	if err != nil {
		panic(err)
	}

	fmt.Println(createdDev)
}
