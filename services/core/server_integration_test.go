package core

import (
	"context"
	"fmt"
	"os"
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

var log = logger.With().Str("test", "core").Logger()
var db *surrealdb.DB
var b *bus.BusConn
var sub *nats.Subscription

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error
	db, b, err = setupConns()
	if err != nil {
		panic(err)
	}
	fmt.Println(" -> connected to bus and db")
	if err := deleteTableOrRecord(db, "devices"); err != nil {
		panic(err)
	}
	fmt.Println(" -> cleaned up db before testing")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	if err := deleteTableOrRecord(db, "devices"); err != nil {
		panic(err)
	}
	fmt.Println(" -> cleaned up db after testing")
	time.Sleep(1 * time.Second)
	b.Drain()
	b.Close()
	db.Close()
	fmt.Println(" -> closed bus and db connection")

	os.Exit(exitVal)
}

// go test -v -timeout 30s -run ^TestPublishDeviceCreate\$ github.com/maxthom/mir/services/core
func TestPublishDeviceCreate(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x994b"
	publishStream := "device.0x944b.core.create.v1alpha"
	devReq := &core.CreateDeviceRequest{
		DeviceId: id,
		Labels: map[string]string{
			"factory": "B",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
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

	dbResp := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, devReq.DeviceId, dbResp[0].Meta.DeviceId)
}

func TestPublishDeviceCreateClient(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x992a"
	devReq := &core.CreateDeviceRequest{
		DeviceId: id,
		Labels: map[string]string{
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	resp, err := PublishDeviceCreateRequest(b, devReq)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, devReq.DeviceId, devRes[0].Meta.DeviceId)
	assert.Equal(t, resp.GetOk().Devices[0].Meta.DeviceId, devRes[0].Meta.DeviceId)
}

func TestPublishDeviceCreateClientNoID(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := ""
	devReq := &core.CreateDeviceRequest{
		DeviceId: id,
		Labels: map[string]string{
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	resp, err := PublishDeviceCreateRequest(b, devReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, resp.GetError() != nil, true)
	assert.Equal(t, resp.GetError().Message, "Invalid device ID")
}

func TestPublishDeviceUpdateTargetIds(t *testing.T) {
	// Arrange
	ctx := context.Background()

	id := "0x777b"
	device := &core.CreateDeviceRequest{
		DeviceId: id,
		Labels: map[string]string{
			"factory": "A",
			"land":    "sheep",
			"owner":   "bob_morrisson",
			"fix":     "cant_be_touch",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	testQuery := &core.UpdateDeviceRequest{
		Targets: &core.Targets{
			Ids: []string{id},
		},
		Request: &core.UpdateDeviceRequest_Meta_{
			Meta: &core.UpdateDeviceRequest_Meta{
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
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := PublishDeviceCreateRequest(b, device); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceUpdateRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})
	if err = deleteDevices(t, db, []string{id}); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, testQuery.Targets.Ids[0], dbResp[0].Meta.DeviceId)
	assert.Equal(t, resp.GetOk().Devices[0].Meta.DeviceId, id)
}

func TestPublishDeviceUpdateTargetLabels(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.UpdateDeviceRequest{
		Targets: &core.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Request: &core.UpdateDeviceRequest_Meta_{
			Meta: &core.UpdateDeviceRequest_Meta{
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
					"mir/device/description": {
						Value: strRef("hello world of devices !"),
					},
				},
			},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "air_quality",
				"mir/device/description": "hello world of devices !",
			},
		},
		{
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "air_quality",
				"mir/device/description": "hello world of devices !",
			},
		},
		{
			DeviceId: deviceIds[2],
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "air_quality",
				"mir/device/description": "hello world of devices !",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := PublishDeviceUpdateRequest(b, testQuery); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	for _, dev := range devRes {
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, *dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[1]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, *dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[2]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "cow")
			assert.Equal(t, *dev.Meta.Labels["fix"], "cant_be_touch")
		}
	}
}

func TestPublishDeviceUpdateTargetAnno(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x14s8c", "0x3499d", "0x16822f"}
	testQuery := &core.UpdateDeviceRequest{
		Targets: &core.Targets{
			Annotations: map[string]string{
				"utility": "hvac",
			},
		},
		Request: &core.UpdateDeviceRequest_Meta_{
			Meta: &core.UpdateDeviceRequest_Meta{
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
			},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "hvac",
				"mir/device/description": "hello world of devices !",
			},
		},
		{
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "hvac",
				"mir/device/description": "hello world of devices !",
			},
		},
		{
			DeviceId: deviceIds[2],
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility":                "humidity",
				"mir/device/description": "hello world of devices !",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := PublishDeviceUpdateRequest(b, testQuery); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	for _, dev := range devRes {
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, *dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[1]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, *dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[2]:
			assert.Equal(t, *dev.Meta.Labels["factory"], "D")
			assert.Equal(t, *dev.Meta.Labels["land"], "cow")
			assert.Equal(t, *dev.Meta.Labels["fix"], "cant_be_touch")
		}
	}
}

func TestPublishDeviceUpdateTargetMixs(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.UpdateDeviceRequest{
		Targets: &core.Targets{
			Ids: []string{deviceIds[2], deviceIds[0]},
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Request: &core.UpdateDeviceRequest_Meta_{
			Meta: &core.UpdateDeviceRequest_Meta{
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
			},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
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
			DeviceId: deviceIds[2],
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
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := PublishDeviceUpdateRequest(b, testQuery); err != nil {
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
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Meta.Labels["factory"], "D")
			assert.Equal(t, dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[1]:
			assert.Equal(t, dev.Meta.Labels["factory"], "D")
			assert.Equal(t, dev.Meta.Labels["land"], "sheep")
			assert.Equal(t, dev.Meta.Labels["fix"], "mazda3sport")
		case deviceIds[2]:
			assert.Equal(t, dev.Meta.Labels["factory"], "D")
			assert.Equal(t, dev.Meta.Labels["land"], "cow")
			assert.Equal(t, dev.Meta.Labels["fix"], "mazda3sport")
		}
	}
}

func TestPublishDeviceDeleteTargetIds(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.DeleteDeviceRequest{
		Targets: &core.Targets{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
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
			DeviceId: deviceIds[2],
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
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err := PublishDeviceDeleteRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	// TODO adjust when delete return list of devices properly
	//assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Equal(t, dbResp[0].Meta.DeviceId, deviceIds[2])
}

func TestPublishDeviceDeleteTargetLabels(t *testing.T) {
	// Arrange
	ctx := context.Background()

	testQuery := &core.DeleteDeviceRequest{
		Targets: &core.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "plane",
			},
		},
	}

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "plane",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "plane",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId: deviceIds[2],
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
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err := PublishDeviceDeleteRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	// TODO adjust when delete return list of devices properly
	//assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Equal(t, dbResp[0].Meta.DeviceId, deviceIds[2])
}

func TestPublishDeviceListTargetIds(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	deviceIds := []string{"0x998c", "0x999d", "0x122f"}
	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}

	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
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
			DeviceId: deviceIds[2],
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
	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResp := executeTestQueryForType[[]Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Check(t, resp.GetOk().Devices[0].Meta.DeviceId == deviceIds[0] || resp.GetOk().Devices[0].Meta.DeviceId == deviceIds[1])
	assert.Check(t, resp.GetOk().Devices[1].Meta.DeviceId == deviceIds[0] || resp.GetOk().Devices[1].Meta.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetLabels(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "lamb",
			},
		},
	}

	deviceIds := []string{"0x12238c", "0x3429d", "0x12cd2f"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "lamb",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "lamb",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
		{
			DeviceId: deviceIds[2],
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
	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResult := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	assert.Equal(t, len(dbResult), 3)
	assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Check(t, resp.GetOk().Devices[0].Meta.DeviceId == deviceIds[0] || resp.GetOk().Devices[0].Meta.DeviceId == deviceIds[1])
	assert.Check(t, resp.GetOk().Devices[1].Meta.DeviceId == deviceIds[0] || resp.GetOk().Devices[1].Meta.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetAnnotations(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{
			Annotations: map[string]string{
				"utility": "air_quality",
			},
		},
	}

	deviceIds := []string{"0x123x", "0x93ef", "0x378a"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "humidity",
			},
		},
		{
			DeviceId: deviceIds[2],
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "humidity",
			},
		},
	}

	// Act
	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dbResult := executeTestQueryForType[[]core.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE meta.deviceId = $id1 OR meta.deviceId = $id2 OR meta.deviceId = $id3;",
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
	assert.Equal(t, len(dbResult), 3)
	assert.Equal(t, len(resp.GetOk().Devices), 1)
	assert.Check(t, resp.GetOk().Devices[0].Meta.DeviceId == deviceIds[0])
}

func TestPublishDeviceListNoTarget(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{},
	}

	deviceIds := []string{"0x123x", "0x93ef", "0x378a"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "humidity",
			},
		},
		{
			DeviceId: deviceIds[2],
			Labels: map[string]string{
				"factory": "D",
				"land":    "cow",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "humidity",
			},
		},
	}

	// Act
	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, len(resp.GetOk().Devices) >= 3, true)
}

func TestCreatedDeviceAlreadyExist(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	deviceIds := []string{"0x666x", "0x666x"}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
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
			DeviceId: deviceIds[1],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "humidity",
			},
		},
	}

	// Act
	resp, err := createDevices(b, devices)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, len(resp), 2)
	assert.Equal(t, resp[1].GetError().Message, "a device with the same id already exists")
}

func TestUpdateNoTargetMetafield(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.UpdateDeviceRequest{}

	// Act
	resp, err := PublishDeviceUpdateRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Act

	// Assert
	assert.Equal(t, resp.GetError().Message, "no target provided for update")
}

func TestDeleteNoTargetMetafield(t *testing.T) {
	// Arrange
	ctx := context.Background()

	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	testQuery := &core.DeleteDeviceRequest{}

	// Act
	resp, err := PublishDeviceDeleteRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Act

	// Assert
	assert.Equal(t, resp.GetError().Message, "no target provided for delete")
}

func TestPublishHearthbeatRequest(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x14ffea"}
	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{
			Ids: deviceIds,
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "hvac",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if err := PublishHearthbeatRequest(b, deviceIds[0]); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devRes, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	if err := deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}

	// Assert
	for _, dev := range devRes.GetOk().Devices {
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
}

// go test -v -timeout 90s -run ^TestDeviceGoesOffline\$ github.com/maxthom/mir/services/core
func TestDeviceGoesOffline(t *testing.T) {
	// Arrange
	ctx := context.Background()

	deviceIds := []string{"0x14aaea"}
	testQuery := &core.ListDeviceRequest{
		Targets: &core.Targets{
			Ids: deviceIds,
		},
	}
	devices := []*core.CreateDeviceRequest{
		{
			DeviceId: deviceIds[0],
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "hvac",
			},
		},
	}

	// Act
	regSrv := NewCore(log, b, db)
	go func() {
		regSrv.Listen(ctx)
	}()

	if _, err := createDevices(b, devices); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	hbTime := time.Now().UTC()
	if err := PublishHearthbeatRequest(b, deviceIds[0]); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	devResOn, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(60 * time.Second)

	devResOff, err := PublishDeviceListRequest(b, testQuery)
	if err != nil {
		t.Error(err)
	}
	if err := deleteDevices(t, db, deviceIds); err != nil {
		t.Error(err)
	}
	// Assert
	for _, dev := range devResOn.GetOk().Devices {
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, hbTime.Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
	for _, dev := range devResOff.GetOk().Devices {
		switch dev.Meta.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, false)
			devTs := AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() > 30, true)
		}
	}
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

	q := "DELETE FROM type::table($tb) WHERE meta.deviceId = \""
	q += strings.Join(ids, "\" OR device_id = \"")
	q += "\";"
	executeTestQueryForType[[]core.Device](t, db,
		q, map[string]string{
			"tb": "devices",
		})
	return nil
}

func createDevices(bus *bus.BusConn, devices []*core.CreateDeviceRequest) ([]*core.CreateDeviceResponse, error) {
	responses := []*core.CreateDeviceResponse{}
	for _, dev := range devices {
		resp, err := PublishDeviceCreateRequest(bus, dev)
		responses = append(responses, resp)
		if err != nil {
			return responses, err
		}
	}
	return responses, nil
}

func deleteTableOrRecord(db *surrealdb.DB, thing string) error {
	if _, err := db.Delete(thing); err != nil {
		return err
	}
	return nil
}

func setupConns() (*surrealdb.DB, *bus.BusConn, error) {
	// Database
	db, err := surrealdb.New("ws://127.0.0.1:8000/rpc")
	if err != nil {
		return db, nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": "root",
		"pass": "root",
	}); err != nil {
		return db, nil, err
	}

	if _, err = db.Use("global", "mir"); err != nil {
		return db, nil, err
	}

	// Bus
	b, err := bus.New("nats://127.0.0.1:4222")
	if err != nil {
		return nil, nil, err
	}

	return db, b, nil
}

func strRef(s string) *string {
	return &s
}
