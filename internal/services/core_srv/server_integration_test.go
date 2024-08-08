package core_srv

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
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
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println("Test Setup")

	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir")
	b = test_utils.SetupNatsConPanic(busUrl)
	coreSrv := NewCore(log, b, db)
	go func() {
		coreSrv.Listen(ctx)
	}()
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "core",
	})
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "core",
	})
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	cancel()
	b.Close()
	db.Close()
	fmt.Println(" -> core")
	fmt.Println(" -> nats")
	fmt.Println(" -> db")

	os.Exit(exitVal)
}

// go test -v -timeout 30s -run ^TestPublishDeviceCreate\$ github.com/maxthom/mir/services/core
func TestPublishDeviceCreate(t *testing.T) {
	// Arrange
	id := "device_create_raw"
	publishStream := "device." + id + ".core.v1alpha.create"
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "core",
			"factory": "B",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Act
	bReq, err := proto.Marshal(reqCreate)
	if err != nil {
		t.Error(err)
	}
	err = b.Publish(publishStream, bReq)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	} else if respList.GetError() != nil {
		t.Error(respList.GetError())
	}

	// Assert
	assert.Equal(t, reqCreate.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
}

func TestPublishDeviceCreateClient(t *testing.T) {
	// Arrange
	id := "device_create"
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "core",
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Subscribe to device created event
	count := 0
	s, err := b.Subscribe(
		core_client.DeviceCreatedEvent.WithId(id),
		func(msg *nats.Msg) {
			count += 1
			msg.Ack()
		})

	// Act
	respCreate, err := core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	} else if respList.GetError() != nil {
		t.Error(respList.GetError())
	}

	// Assert
	assert.Equal(t, reqCreate.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
	assert.Equal(t, respCreate.GetOk().Devices[0].Spec.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
	assert.Equal(t, 1, count)
	s.Unsubscribe()
}

func TestPublishDeviceCreateClientNoID(t *testing.T) {
	// Arrange
	id := ""
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "core",
			"factory": "A",
			"model":   "xx021",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	// Act
	respCreate, err := core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, respCreate.GetError() != nil, true)
	assert.Equal(t, respCreate.GetError().Message, "Invalid device ID")
}

func TestPublishDeviceUpdateTargetIds(t *testing.T) {
	// Arrange
	id := "device_update_target_ids"
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "core",
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

	// Subscribe to device updated event
	count := 0
	s, err := b.Subscribe(
		core_client.DeviceUpdatedEvent.WithId(id),
		func(msg *nats.Msg) {
			count += 1
			msg.Ack()
		})

	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{id},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
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
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
				"utility": {
					Value: strRef("major"),
				},
				"instance": nil,
				"deploy": {
					Value: nil,
				},
			},
		},
	}

	// Act
	if _, err := core_client.PublishDeviceCreateRequest(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respUpd, err := core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})

	// Assert
	assert.Equal(t, reqUpd.Targets.Ids[0], respDb[0].Spec.DeviceId)
	assert.Equal(t, respUpd.GetOk().Devices[0].Spec.DeviceId, id)
	assert.Equal(t, 1, count)
	s.Unsubscribe()
}

func TestPublishDeviceUpdateTargetNames(t *testing.T) {
	// Arrange
	id := "device_update_target_names"
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "core",
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

	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Names: []string{id},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
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
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
				"utility": {
					Value: strRef("major"),
				},
				"instance": nil,
				"deploy": {
					Value: nil,
				},
			},
		},
	}

	// Act
	if _, err := core_client.PublishDeviceCreateRequest(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respUpd, err := core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})

	// Assert
	assert.Equal(t, reqUpd.Targets.Names[0], respDb[0].Spec.DeviceId)
	assert.Equal(t, respUpd.GetOk().Devices[0].Meta.Name, id)
}

func TestPublishDeviceUpdateTargetNamespace(t *testing.T) {
	// Arrange
	id := "device_update_target_namespace"
	ns := "testing_" + id
	reqCreate := &core_api.CreateDeviceRequest{
		DeviceId:  id,
		Namespace: ns,
		Labels: map[string]string{
			"testing": "core",
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

	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Namespaces: []string{ns},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
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
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
				"utility": {
					Value: strRef("major"),
				},
				"instance": nil,
				"deploy": {
					Value: nil,
				},
			},
		},
	}

	// Act
	if _, err := core_client.PublishDeviceCreateRequest(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respUpd, err := core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id;",
		map[string]string{
			"tb": "devices",
			"id": id,
		})

	// Assert
	assert.Equal(t, reqUpd.Targets.Namespaces[0], respDb[0].Meta.Namespace)
	assert.Equal(t, respUpd.GetOk().Devices[0].Spec.DeviceId, id)
}

func TestPublishDeviceUpdateTargetLabels(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_update_target_labels_1", "device_update_target_labels_2", "device_update_target_labels_3"}
	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
				"owner": {
					Value: nil,
				},
				"fix": {
					Value: strRef("mazda3sport"),
				},
			},
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
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
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := core_client.PublishDeviceUpdateRequest(b, reqUpd); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	for _, dev := range respDb {
		switch dev.Spec.DeviceId {
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
	deviceIds := []string{"device_update_target_anno_1", "device_update_target_anno_2", "device_update_target_anno_3"}
	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Annotations: map[string]string{
				"utility": "hvac",
			},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
				"owner": {
					Value: nil,
				},
				"fix": {
					Value: strRef("mazda3sport"),
				},
			},
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
				"utility": {
					Value: nil,
				},
				"instance": nil,
				"deploy": {
					Value: strRef("in_hell"),
				},
			},
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := core_client.PublishDeviceUpdateRequest(b, reqUpd); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	for _, dev := range respDb {
		switch dev.Spec.DeviceId {
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
	deviceIds := []string{"device_update_target_mix_1", "device_update_target_mix_2", "device_update_target_mix_3"}
	reqUpd := &core_api.UpdateDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{deviceIds[2], deviceIds[0]},
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Labels: map[string]*core_api.UpdateDeviceRequest_OptString{
				"owner": {
					Value: nil,
				},
				"fix": {
					Value: strRef("mazda3sport"),
				},
			},
			Annotations: map[string]*core_api.UpdateDeviceRequest_OptString{
				"utility": {
					Value: nil,
				},
				"instance": nil,
				"deploy": {
					Value: strRef("in_hell"),
				},
			},
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if _, err := core_client.PublishDeviceUpdateRequest(b, reqUpd); err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]core_api.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE device_id = $id1 OR device_id = $id2 OR device_id = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	for _, dev := range respDb {
		switch dev.Spec.DeviceId {
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
	deviceIds := []string{"device_delete_target_ids_1", "device_delete_target_ids_2", "device_delete_target_ids_3"}
	reqDel := &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
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

	// Subscribe to device deleted event
	count := 0
	s, err := b.Subscribe(
		core_client.DeviceDeletedEvent.WithId("*"),
		func(msg *nats.Msg) {
			count += 1
			msg.Ack()
		})

	// Act
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 1)
	assert.Equal(t, respDb[0].Spec.DeviceId, deviceIds[2])
	assert.Equal(t, 2, count)
	s.Unsubscribe()
}

func TestPublishDeviceDeleteTargetNames(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_delete_target_ids_1", "device_delete_target_ids_2", "device_delete_target_ids_3"}
	reqDel := &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Names: []string{deviceIds[0], deviceIds[1]},
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Name:      deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Name:      deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Name:      deviceIds[2],
			Namespace: "testing_core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err := core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 1)
	// TODO adjust when delete return list of devices properly
	//assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Equal(t, respDb[0].Spec.DeviceId, deviceIds[2])
}

func TestPublishDeviceDeleteTargetNamespace(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_delete_target_ns_1", "device_delete_target_ns_2", "device_delete_target_ns_3"}
	ns := "testing_" + strings.Join(deviceIds, "_")
	reqDel := &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Namespaces: []string{ns},
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: ns,
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: ns,
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: ns,
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err := core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}

	// Wait for written to db
	time.Sleep(1 * time.Second)

	dbResp := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(dbResp), 0)
	// TODO adjust when delete return list of devices properly
	//assert.Equal(t, len(resp.GetOk().Devices), 2)
	//assert.Equal(t, dbResp[0].spec.deviceId, deviceIds[2])
}

func TestPublishDeviceDeleteTargetLabels(t *testing.T) {
	// Arrange
	reqDel := &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "plane",
			},
		},
	}

	deviceIds := []string{"device_delete_target_lbls_1", "device_delete_target_lbls_2", "device_delete_target_lbls_3"}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err := core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 1)
	// TODO adjust when delete return list of devices properly
	//assert.Equal(t, len(resp.GetOk().Devices), 2)
	assert.Equal(t, respDb[0].Spec.DeviceId, deviceIds[2])
}

func TestPublishDeviceListTargetIds(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_list_target_ids_1", "device_list_target_ids_2", "device_list_target_ids_3"}
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}

	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 3)
	assert.Equal(t, len(respList.GetOk().Devices), 2)
	assert.Check(t, respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[1])
	assert.Check(t, respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetNames(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_list_target_names_1", "device_list_target_names_2", "device_list_target_names_3"}
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Names: []string{deviceIds[0], deviceIds[1]},
		},
	}

	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Name:      deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Name:      deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Name:      deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 3)
	assert.Equal(t, len(respList.GetOk().Devices), 2)
	assert.Check(t, respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[1])
	assert.Check(t, respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetNamespace(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_list_target_ns_1", "device_list_target_ns_2", "device_list_target_ns_3"}
	ns := "testing_" + strings.Join(deviceIds, "_")
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Namespaces: []string{ns},
		},
	}

	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: ns,
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: ns,
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: ns + "cacaouette",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]mir_models.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 3)
	assert.Equal(t, len(respList.GetOk().Devices), 2)
	assert.Check(t, respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[1])
	assert.Check(t, respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetLabels(t *testing.T) {
	// Arrange
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Labels: map[string]string{
				"factory": "D",
				"land":    "lamb",
			},
		},
	}

	deviceIds := []string{"device_list_target_lbls_1", "device_list_target_lbls_2", "device_list_target_lbls_3"}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]core_api.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 3)
	assert.Equal(t, len(respList.GetOk().Devices), 2)
	assert.Check(t, respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[1])
	assert.Check(t, respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[0] || respList.GetOk().Devices[1].Spec.DeviceId == deviceIds[1])
}

func TestPublishDeviceListTargetAnnotations(t *testing.T) {
	// Arrange
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Annotations: map[string]string{
				"utility": "air_quality_target_anno",
			},
		},
	}

	deviceIds := []string{"device_list_target_anno_1", "device_list_target_anno_2", "device_list_target_anno_3"}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"factory": "D",
				"land":    "sheep",
				"owner":   "bob_morrisson",
				"fix":     "cant_be_touch",
			},
			Annotations: map[string]string{
				"utility": "air_quality_target_anno",
			},
		},
		{
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respDb := test_utils.ExecuteTestQueryForType[[]core_api.Device](t, db,
		"SELECT * FROM type::table($tb) WHERE spec.deviceId = $id1 OR spec.deviceId = $id2 OR spec.deviceId = $id3;",
		map[string]string{
			"tb":  "devices",
			"id1": deviceIds[0],
			"id2": deviceIds[1],
			"id3": deviceIds[2],
		})

	// Assert
	assert.Equal(t, len(respDb), 3)
	assert.Equal(t, len(respList.GetOk().Devices), 1)
	assert.Check(t, respList.GetOk().Devices[0].Spec.DeviceId == deviceIds[0])
}

func TestPublishDeviceListNoTarget(t *testing.T) {
	// Arrange
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{},
	}

	deviceIds := []string{"device_list_target_no_1", "device_list_target_no_2", "device_list_target_no_3"}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[2],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, len(respList.GetOk().Devices) >= 3, true)
}

func TestCreatedDeviceAlreadyExist(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_already_exist_1", "device_already_exist_1"}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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
			DeviceId:  deviceIds[1],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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

	// Subscribe to device created event
	count := 0
	s, err := b.Subscribe(
		core_client.DeviceCreatedEvent.WithId(deviceIds[0]),
		func(msg *nats.Msg) {
			count += 1
			msg.Ack()
		})

	// Act
	respCreate, err := test_utils.CreateDevices(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, len(respCreate), 2)
	assert.Equal(t, respCreate[1].GetError().Message, "a device with the same id already exists")

	assert.Equal(t, 1, count) // We create two devices, so only second one is not working
	s.Unsubscribe()
}

func TestUpdateNoTargetMetafield(t *testing.T) {
	// Arrange
	reqUpd := &core_api.UpdateDeviceRequest{}

	// Act
	respUpd, err := core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, respUpd.GetError().Message, "no target provided for update")
}

func TestDeleteNoTargetMetafield(t *testing.T) {
	// Arrange
	reqDel := &core_api.DeleteDeviceRequest{}

	// Act
	respDel, err := core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, respDel.GetError().Message, "no target provided for delete")
}

func TestDeviceGoesOnline(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_goes_online"}
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: deviceIds,
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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

	// Subscribe to device online event
	onlineEventCount := 0
	s, err := b.Subscribe(
		core_client.DeviceOnlineEvent.WithId(deviceIds[0]),
		func(msg *nats.Msg) {
			onlineEventCount += 1
			msg.Ack()
		})

	// Act
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	if err := core_client.PublishHearthbeatStream(b, deviceIds[0]); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}

	// Assert
	for _, dev := range respList.GetOk().Devices {
		switch dev.Spec.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
	assert.Equal(t, 1, onlineEventCount)
	s.Unsubscribe()
}

// go test -v -timeout 90s -run ^TestDeviceGoesOffline\$ github.com/maxthom/mir/services/core
func TestDeviceGoesOffline(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_goes_offline"}
	reqList := &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: deviceIds,
		},
	}
	reqCreate := []*core_api.CreateDeviceRequest{
		{
			DeviceId:  deviceIds[0],
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
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

	// Subscribe to device offline event
	offlineEventCount := 0
	s, err := b.Subscribe(
		core_client.DeviceOfflineEvent.WithId(deviceIds[0]),
		func(msg *nats.Msg) {
			offlineEventCount += 1
			msg.Ack()
		})

	// Act
	if _, err := test_utils.CreateDevices(b, reqCreate); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	hbTime := time.Now().UTC()
	if err := core_client.PublishHearthbeatStream(b, deviceIds[0]); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respListOn, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(60 * time.Second)

	respListOff, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}

	// Assert
	for _, dev := range respListOn.GetOk().Devices {
		switch dev.Spec.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, hbTime.Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
	for _, dev := range respListOff.GetOk().Devices {
		switch dev.Spec.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, false)
			devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() > 30, true)
		}
	}

	assert.Equal(t, 1, offlineEventCount)
	s.Unsubscribe()
}

func strRef(s string) *string {
	return &s
}
