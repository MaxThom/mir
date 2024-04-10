package registration

import (
	"context"
	"testing"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/registration"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"gotest.tools/assert"
)

var log = logger.With().Str("test", "registration").Logger()

// go test -v -timeout 30s -run ^TestPublishDeviceCreateSuccess\$ github.com/maxthom/mir/services/registration
func TestPublishDeviceCreateSuccess(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	db, bus, sub, err := setupConns(bus.DeviceStreamSubject)
	t.Cleanup(func() {
		db.Close()
		cancel()
	})
	if err != nil {
		t.Error(err)
	}

	publishStream := "test.v1alpha.device.create"
	devReq := &registration.CreateDeviceRequest{
		DeviceId:    "0x994b",
		Description: "hello world of devices !",
		Labels: map[string]string{
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility": "air_quality",
		},
	}

	// Act
	regSrv := NewRegistrationServer(log, bus, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	bReq, err := proto.Marshal(devReq)
	if err != nil {
		t.Error(err)
	}
	err = bus.Publish(publishStream, bReq)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	devRes := executeQueryForType[[]registration.CreateDeviceRequest](t, db,
		"SELECT * FROM type::table($tb);",
		map[string]string{
			"tb": "devices",
		})
	deleteTableOrRecord(t, db, "devices")

	// Assert
	assert.Equal(t, devReq.DeviceId, devRes[0].DeviceId)
	assert.Equal(t, devReq.Description, devRes[0].Description)
}

func TestPublishDeviceCreateClientSuccess(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	db, bus, sub, err := setupConns(bus.DeviceStreamSubject)
	t.Cleanup(func() {
		db.Close()
		cancel()
	})
	if err != nil {
		t.Error(err)
	}

	devReq := &registration.CreateDeviceRequest{
		DeviceId:    "0x994b",
		Description: "hello world of devices !",
		Labels: map[string]string{
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility": "air_quality",
		},
	}

	// Act
	regSrv := NewRegistrationServer(log, bus, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	resp, err := PublishDeviceCreateRequest(ctx, bus, devReq)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	devRes := executeQueryForType[[]registration.CreateDeviceRequest](t, db,
		"SELECT * FROM type::table($tb);",
		map[string]string{
			"tb": "devices",
		})
	deleteTableOrRecord(t, db, "devices")

	// Assert
	assert.Equal(t, devReq.DeviceId, devRes[0].DeviceId)
	assert.Equal(t, devReq.Description, devRes[0].Description)
	assert.Equal(t, resp.DeviceId, devRes[0].DeviceId)
	assert.Equal(t, resp.Msg, "Device created")
}

func executeQueryForType[T any](t *testing.T, db *surrealdb.DB, query string, vars map[string]string) T {
	result, err := db.Query(query, vars)
	if err != nil {
		t.Error(err)
	}

	res, err := surrealdb.SmartUnmarshal[T](result, err)
	if err != nil {
		t.Error(err)
	}

	return res
}

func deleteTableOrRecord(t *testing.T, db *surrealdb.DB, thing string) {
	if _, err := db.Delete(thing); err != nil {
		t.Error(err)
	}
}

func setupJetStreamConns(ctx context.Context, jsCfg jetstream.StreamConfig, consCfg jetstream.ConsumerConfig) (*surrealdb.DB, *bus.BusConn, jetstream.Consumer, error) {
	// Database
	db, err := surrealdb.New("ws://127.0.0.1:8000/rpc")
	if err != nil {
		return db, nil, nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": "root",
		"pass": "root",
	}); err != nil {
		return db, nil, nil, err
	}

	// Bus
	b, _, cons, err := createPublisherForStream(ctx, "nats://127.0.0.1:4222", jsCfg, consCfg)
	if err != nil {
		return db, b, cons, err
	}

	return db, b, cons, nil
}

func createPublisherForStream(ctx context.Context, busUrl string, jsCfg jetstream.StreamConfig, consCfg jetstream.ConsumerConfig) (*bus.BusConn, jetstream.Stream, jetstream.Consumer, error) {
	b, err := bus.New(busUrl)
	if err != nil {
		return nil, nil, nil, err
	}

	js, err := jetstream.New(b.Conn)
	if err != nil {
		return b, nil, nil, err
	}

	stream, err := js.CreateOrUpdateStream(ctx, jsCfg)
	if err != nil {
		return b, stream, nil, err
	}

	// retrieve consumer handle from a stream
	cons, err := stream.CreateOrUpdateConsumer(ctx, consCfg)

	return b, stream, cons, err
}

func setupConns(subject string) (*surrealdb.DB, *bus.BusConn, *nats.Subscription, error) {
	// Database
	db, err := surrealdb.New("ws://127.0.0.1:8000/rpc")
	if err != nil {
		return db, nil, nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": "root",
		"pass": "root",
	}); err != nil {
		return db, nil, nil, err
	}

	// Bus
	b, err := bus.New("nats://127.0.0.1:4222")
	if err != nil {
		return nil, nil, nil, err
	}

	sub, err := b.SubscribeSync(subject)
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to subject")
	}

	return db, b, sub, nil
}
