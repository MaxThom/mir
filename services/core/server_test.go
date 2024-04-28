package core

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/nats-io/nats.go"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"gotest.tools/assert"
)

// TODO make use of the t.SubTest thingy to better handle cleaning such as
// db connection closing
// perhaps also create a large set of data that can be manipulated throughtout
// all the test and deleted at the end
var log = logger.With().Str("test", "core").Logger()
var db *surrealdb.DB
var b *bus.BusConn
var sub *nats.Subscription

func init() {
	var err error
	db, b, sub, err = setupConns(bus.DeviceStreamSubject)
	if err != nil {
		panic(err)
	}
}

// go test -v -timeout 30s -run ^TestPublishDeviceCreateSuccess\$ github.com/maxthom/mir/services/core
func TestPublishDeviceCreateSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x994b"
	publishStream := "test.v1alpha.device.create"
	devReq := &core.CreateDeviceRequest{
		DeviceId:    id,
		Description: "hello world of devices !",
		Labels: map[string]string{
			"factory": "B",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility": "air_quality",
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	bReq, err := proto.Marshal(devReq)
	if err != nil {
		t.Error(err)
	}
	err = b.Publish(publishStream, bReq)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.CreateDeviceRequest](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, devReq.DeviceId, dbResp[0].DeviceId)
	assert.Equal(t, devReq.Description, dbResp[0].Description)
}

func TestPublishDeviceCreateClientSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x992a"
	devReq := &core.CreateDeviceRequest{
		DeviceId:    id,
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
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	resp, err := PublishDeviceCreateRequest(ctx, b, devReq)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes := executeTestQueryForType[[]core.CreateDeviceRequest](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, devReq.DeviceId, devRes[0].DeviceId)
	assert.Equal(t, devReq.Description, devRes[0].Description)
	assert.Equal(t, resp.DeviceId, devRes[0].DeviceId)
	assert.Equal(t, resp.Msg[0], "Device created")
}

func TestPublishDeviceUpdateTargetIdsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x777b"
	device := &core.CreateDeviceRequest{
		DeviceId:    id,
		Description: "hello world of devices !",
		Labels: map[string]string{
			"factory": "A",
			"land":    "sheep",
			"owner":   "bob_morrisson",
			"fix":     "cant_be_touch",
		},
		Annotations: map[string]string{
			"utility": "air_quality",
		},
	}

	testQuery := &core.UpdateDeviceRequest{
		TargetIds:    []string{id},
		TargetLabels: map[string]string{},
		Description:  strRef("yiihayy"),
		Labels: map[string]*core.UpdateDeviceRequest_OptString{
			"factory": {
				Value: strRef("site_b"),
			},
			"land": nil,
			"owner": {
				Value: nil,
			},
			"model": {
				Value: strRef("mazda3sport"),
			},
		},
		Annotations: map[string]*core.UpdateDeviceRequest_OptString{
			"utility": {
				Value: strRef("major"),
			},
			"instance": nil,
			"deploy": {
				Value: nil,
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := PublishDeviceCreateRequest(ctx, b, device); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceUpdateRequest(ctx, b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.CreateDeviceRequest](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, testQuery.TargetIds[0], dbResp[0].DeviceId)
	assert.Equal(t, *testQuery.Description, dbResp[0].Description)
	assert.Equal(t, resp.AffectedDevices[0], id)
}

func TestPublishDeviceUpdateTargetLabelsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.UpdateDeviceRequest{
		TargetIds: []string{},
		TargetLabels: map[string]string{
			"factory": "D",
			"land":    "sheep",
		},
		Description: strRef("yiihayy"),
		Labels: map[string]*core.UpdateDeviceRequest_OptString{
			"owner": {
				Value: nil,
			},
			"fix": {
				Value: strRef("mazda3sport"),
			},
		},
		Annotations: map[string]*core.UpdateDeviceRequest_OptString{
			"utility": {
				Value: nil,
			},
			"instance": nil,
			"deploy": {
				Value: strRef("in_hell"),
			},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := PublishDeviceUpdateRequest(ctx, b, testQuery); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": "0x998c",
			"id2": "0x999d",
			"id3": "0x122f",
		})
	if err := deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	for _, dev := range devRes {
		switch dev.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "sheep")
			assert.Equal(t, dev.Labels["fix"], "mazda3sport")
		case deviceIds[1]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "sheep")
			assert.Equal(t, dev.Labels["fix"], "mazda3sport")
		case deviceIds[2]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "cow")
			assert.Equal(t, dev.Labels["fix"], "cant_be_touch")
		}
	}
}

func TestPublishDeviceUpdateTargetMixsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.UpdateDeviceRequest{
		TargetIds: []string{deviceIds[2], deviceIds[0]},
		TargetLabels: map[string]string{
			"factory": "D",
			"land":    "sheep",
		},
		Description: strRef("yiihayy"),
		Labels: map[string]*core.UpdateDeviceRequest_OptString{
			"owner": {
				Value: nil,
			},
			"fix": {
				Value: strRef("mazda3sport"),
			},
		},
		Annotations: map[string]*core.UpdateDeviceRequest_OptString{
			"utility": {
				Value: nil,
			},
			"instance": nil,
			"deploy": {
				Value: strRef("in_hell"),
			},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := PublishDeviceUpdateRequest(ctx, b, testQuery); err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})
	if err := deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	for _, dev := range dbResp {
		switch dev.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "sheep")
			assert.Equal(t, dev.Labels["fix"], "mazda3sport")
		case deviceIds[1]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "sheep")
			assert.Equal(t, dev.Labels["fix"], "mazda3sport")
		case deviceIds[2]:
			assert.Equal(t, dev.Labels["factory"], "D")
			assert.Equal(t, dev.Labels["land"], "cow")
			assert.Equal(t, dev.Labels["fix"], "mazda3sport")
		}
	}
}

func TestPublishDeviceDeleteTargetIdsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.DeleteDeviceRequest{
		TargetIds: []string{deviceIds[0], deviceIds[1]},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceDeleteRequest(ctx, b, testQuery)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})
	if err = deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dbResp), 1)
	assert.Equal(t, len(resp.AffectedDevices), 2)
	assert.Equal(t, dbResp[0].DeviceId, deviceIds[2])
}

func TestPublishDeviceDeleteTargetLabelsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	testQuery := &core.DeleteDeviceRequest{
		TargetLabels: map[string]string{
			"factory": "D",
			"land":    "sheep",
		},
	}

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceDeleteRequest(ctx, b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	if err = deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dbResp), 1)
	assert.Equal(t, len(resp.AffectedDevices), 2)
	assert.Equal(t, dbResp[0].DeviceId, deviceIds[2])
}

func TestPublishDeviceListTargetIdsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.ListDeviceRequest{
		TargetIds: []string{deviceIds[0], deviceIds[1]},
	}

	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(ctx, b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})
	if err = deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dbResp), 3)
	assert.Equal(t, len(resp.Devices), 2)
	assert.Check(t, resp.Devices[0].DeviceId == deviceIds[0] || resp.Devices[0].DeviceId == deviceIds[1])
	assert.Check(t, resp.Devices[1].DeviceId == deviceIds[0] || resp.Devices[1].DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetLabelsSuccess(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, sub, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.ListDeviceRequest{
		TargetLabels: map[string]string{
			"factory": "D",
			"land":    "sheep",
		},
	}

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId:    deviceIds[0],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[1],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId:    deviceIds[2],
			Description: "hello world of devices !",
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	// Act
	if err := createDevices(ctx, b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(ctx, b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResult := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": "0x998c",
			"id2": "0x999d",
			"id3": "0x122f",
		})

	// Assert
	assert.Equal(t, len(dbResult), 3)
	assert.Equal(t, len(resp.Devices), 2)
	assert.Check(t, resp.Devices[0].DeviceId == "0x998c" || resp.Devices[0].DeviceId == "0x999d")
	assert.Check(t, resp.Devices[1].DeviceId == "0x998c" || resp.Devices[1].DeviceId == "0x999d")
}

func executeTestQueryForType[T any](t *testing.T, db *surrealdb.DB, query string, vars map[string]string) T {
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

func deleteDevices(t *testing.T, db *surrealdb.DB, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("must provice at least one id")
	}

	q := "DELETE FROM type::table($tb) WHERE device_id = \""
	q += strings.Join(ids, "\" OR device_id = \"")
	q += "\";"
	executeTestQueryForType[[]core.Device](t, db,
		q, map[string]string{
			"tb": "devices",
		})
	return nil
}

func createDevices(ctx context.Context, bus *bus.BusConn, devices []*core.CreateDeviceRequest) error {
	for _, dev := range devices {
		_, err := PublishDeviceCreateRequest(ctx, b, dev)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteTableOrRecord(t *testing.T, db *surrealdb.DB, thing string) {
	if _, err := db.Delete(thing); err != nil {
		t.Error(err)
	}
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

func strRef(s string) *string {
	return &s
}
