package core_srv

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	core_testv1 "github.com/maxthom/mir/internal/servers/core_srv/proto_test/gen/core_test/v1"
	"github.com/maxthom/mir/internal/servers/protocfg_srv"
	"github.com/maxthom/mir/internal/services/schema_cache"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	mirDev "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("core")
var db *surrealdb.DB
var b *bus.BusConn
var sub *nats.Subscription
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error

	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir_testing")
	b = test_utils.SetupNatsConPanic(busUrl)
	mSdk, err = mir.Connect("test_coresrv", busUrl)
	if err != nil {
		panic(err)
	}
	coreSrv, err := NewCore(log, mSdk, mng.NewSurrealMirStore(db))
	if err := coreSrv.Serve(); err != nil {
		panic(err)
	}
	cc, err := schema_cache.NewMirProtoCache(log, mSdk)
	if err != nil {
		panic(err)
	}
	cfgSrv, err := protocfg_srv.NewProtoCfg(log, mSdk, mng.NewSurrealMirStore(db), cc)
	if err := cfgSrv.Serve(); err != nil {
		panic(err)
	}
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
	core_client.PublishDeviceDeleteRequest(b, &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{"device_auto_provision"},
		},
	})
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	coreSrv.Shutdown()
	cfgSrv.Shutdown()
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
	publishStream := "client." + id + ".core.v1alpha.create"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
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

	respList, err := core_client.PublishDeviceListRequest(b, &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	} else if respList.GetError() != "" {
		t.Error(respList.GetError())
	}

	// Assert
	assert.Equal(t, reqCreate.Spec.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
}

func TestPublishDeviceCreateClient(t *testing.T) {
	// Arrange
	id := "device_create"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
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

	respList, err := core_client.PublishDeviceListRequest(b, &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, reqCreate.Spec.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
	assert.Equal(t, respCreate.GetOk().Spec.DeviceId, respList.GetOk().Devices[0].Spec.DeviceId)
	assert.Equal(t, 1, count)
	s.Unsubscribe()
}

func TestPublishDeviceCreateClientNoID(t *testing.T) {
	// Arrange
	id := ""
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	// Act
	resp, _ := core_client.PublishDeviceCreateRequest(b, reqCreate)

	// Assert
	assert.Equal(t, resp.GetError() != "", true)
	assert.Equal(t, resp.GetError(), "error creating device: device name and id are missing")
}

func TestPublishDeviceCreateClientNoNamespace(t *testing.T) {
	// Arrange
	id := "create_dev_no_namespace"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "",
			Labels: map[string]string{
				"testing": "core",
				"factory": "A",
				"model":   "xx021",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	// Act
	respCreate, err := core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, reqCreate.Spec.DeviceId, respCreate.GetOk().Spec.DeviceId)
	assert.Equal(t, respCreate.GetOk().Meta.Namespace, "default")
}

func TestPublishDeviceUpdateTargetIds(t *testing.T) {
	// Arrange
	id := "device_update_target_ids"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
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

	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
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
			Annotations: map[string]*common_apiv1.OptString{
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
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Names: []string{id},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
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
			Annotations: map[string]*common_apiv1.OptString{
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
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
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
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Namespaces: []string{ns},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
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
			Annotations: map[string]*common_apiv1.OptString{
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
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
				"owner": {
					Value: nil,
				},
				"fix": {
					Value: strRef("mazda3sport"),
				},
			},
			Annotations: map[string]*common_apiv1.OptString{
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
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
			assert.Equal(t, dev.Meta.Labels["fix"], "cant_be_touch")
		}
	}
}

func TestPublishDeviceUpdateTargetMixs(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_update_target_mix_1", "device_update_target_mix_2", "device_update_target_mix_3"}
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{deviceIds[2], deviceIds[0]},
			Labels: map[string]string{
				"factory": "D",
				"land":    "sheep",
			},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
				"owner": {
					Value: nil,
				},
				"fix": {
					Value: strRef("mazda3sport"),
				},
			},
			Annotations: map[string]*common_apiv1.OptString{
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
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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

	respDb := test_utils.ExecuteTestQueryForType[[]*core_apiv1.Device](t, db,
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
	reqDel := &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
			},
		},
	}

	// Subscribe to device deleted event
	count := 0
	s, err := b.Subscribe(
		core_client.DeviceDeletedEvent.WithId("*"),
		func(msg *nats.Msg) {
			if slices.Contains(deviceIds, clients.ServerSubject(msg.Subject).GetId()) {
				count += 1
			}
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
	deviceIds := []string{"device_delete_target_names_1", "device_delete_target_names_2", "device_delete_target_names_3"}
	reqDel := &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Names: []string{deviceIds[0], deviceIds[1]},
		},
	}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqDel := &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Namespaces: []string{ns},
		},
	}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[0],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[1],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqDel := &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"factory": "D",
				"land":    "plane",
			},
		},
	}

	deviceIds := []string{"device_delete_target_lbls_1", "device_delete_target_lbls_2", "device_delete_target_lbls_3"}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[0],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[1],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{deviceIds[0], deviceIds[1]},
		},
	}

	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Names: []string{deviceIds[0], deviceIds[1]},
		},
	}

	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Namespaces: []string{ns},
		},
	}

	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[0],
				Namespace: ns,
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[1],
				Namespace: ns,
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"factory": "D",
				"land":    "lamb",
			},
		},
	}

	deviceIds := []string{"device_list_target_lbls_1", "device_list_target_lbls_2", "device_list_target_lbls_3"}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[0],
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "core",
					"factory": "D",
					"land":    "lamb",
					"owner":   "bob_morrisson",
					"fix":     "cant_be_touch",
				},
				Annotations: map[string]string{
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[1],
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "core",
					"factory": "D",
					"land":    "lamb",
					"owner":   "bob_morrisson",
					"fix":     "cant_be_touch",
				},
				Annotations: map[string]string{
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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

	respDb := test_utils.ExecuteTestQueryForType[[]core_apiv1.Device](t, db,
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

func TestPublishDeviceListNoTarget(t *testing.T) {
	// Arrange
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{},
	}

	deviceIds := []string{"device_list_target_no_1", "device_list_target_no_2", "device_list_target_no_3"}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
		{
			Meta: &core_apiv1.Meta{
				Name:      deviceIds[2],
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
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[2],
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
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
		{
			Meta: &core_apiv1.Meta{
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
					"utility":                "air_quality",
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[1],
			},
		},
	}

	// Subscribe to device created event
	count := 0
	s, _ := b.Subscribe(
		core_client.DeviceCreatedEvent.WithId("*"),
		func(msg *nats.Msg) {
			if slices.Contains(deviceIds, clients.ServerSubject(msg.Subject).GetId()) {
				count += 1
			}
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
	assert.Equal(t, respCreate[1].GetError(), "error creating device: device device_already_exist_1/testing_core with deviceId device_already_exist_1 already exist")

	assert.Equal(t, 1, count) // We create two devices, so only second one is not working
	s.Unsubscribe()
}

func TestUpdateNoTargetMetafield(t *testing.T) {
	// Arrange
	reqUpd := &core_apiv1.UpdateDeviceRequest{}

	// Act
	respUpd, err := core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, respUpd.GetError(), "error no target found: No device target provided")
}

func TestDeleteNoTargetMetafield(t *testing.T) {
	// Arrange
	reqDel := &core_apiv1.DeleteDeviceRequest{}

	// Act
	respDel, err := core_client.PublishDeviceDeleteRequest(b, reqDel)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, respDel.GetError(), "error no target found: No device target provided")
}

func TestDeviceCreateDeviceIdAlreadyExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	b := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "create_dev_same_id_1",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"factory": "D",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86cmd",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "create_dev_same_id_2",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"factory": "D",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86cmd",
		},
	})

	// Act
	resp, err := b.Incubate()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, resp[1].GetError(), "error creating device: device create_dev_same_id_2/testing_core with deviceId 0xf86cmd already exist")
}

func TestDeviceCreateDeviceNameNsAlreadyExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	b := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "create_dev_same_id_3",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"factory": "D",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "create_dev_same_id_3",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"factory": "D",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz",
		},
	})

	// Act
	resp, err := b.Incubate()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, resp[1].GetError(), "error creating device: device create_dev_same_id_3/testing_core with deviceId 0xf86xyz already exist")
}

func TestDeviceUpsertDevice(t *testing.T) {
	// Arrange
	id := "0x2312"
	req := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name:      strRef("upsert_device"),
			Namespace: strRef("testing_core"),
			Labels: map[string]*common_apiv1.OptString{
				"testing": {
					Value: strRef("core"),
				},
				"factory": {
					Value: strRef("A"),
				},
			},
		},
		Spec: &core_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: strRef(id),
		},
		Targets: &core_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	}

	count := 0
	_, err := b.Subscribe(
		core_client.DeviceCreatedEvent.WithId(id),
		func(msg *nats.Msg) {
			count += 1
			msg.Ack()
		})

	// Act
	resp, err := core_client.PublishDeviceUpdateRequest(b, req)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 1, count)
	assert.Equal(t, id, resp.GetOk().Devices[0].Spec.DeviceId)
}

func TestDeviceUpdateManyTargetSameDeviceId(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_sameid_1",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "a",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm24",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_sameid_2",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "a",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz24",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Spec: &core_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: strRef("sameid"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "a",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	time.Sleep(2 * time.Second)
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update multiple devices as deviceId must be unique")
}

func TestDeviceUpdateManyTargetSameNameNoExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_samename_noexist_1",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "b",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm21",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_samename_noexist_2",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "b",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz21",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name: strRef("samebloodyname"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "b",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update device as multiple device will have the same name 'samebloodyname' in namespace 'testing_core'")
}

func TestDeviceUpdateManyTargetSameNameOneExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_samename_oneexist",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "c",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm17",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "samebloodyname",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "c",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz17",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name: strRef("samebloodyname"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "c",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update device as multiple device will have the same name 'samebloodyname' in namespace 'testing_core'")
}

func TestDeviceUpdateManyTargetSameNamespaceNoExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_ns_no_exist",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "d",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm12",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_ns_no_exist",
			Namespace: "testing_core_2",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "d",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz12",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Namespace: strRef("samebloodynamespace"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "d",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update device as multiple device will have the same name 'update_dev_ns_no_exist' in namespace 'samebloodynamespace'")
}

func TestDeviceUpdateManyTargetSameNamespaceOneExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_ns_one_exist",
			Namespace: "samebloodynamespace",
			Labels: map[string]string{
				"testing": "core",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm7",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_ns_one_exist",
			Namespace: "testing_core_2",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "e",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz7",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Namespace: strRef("samebloodynamespace"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "e",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update device as name 'update_dev_ns_one_exist' is already in use in namespace 'samebloodynamespace'")
}

func TestDeviceUpdateManyTargetSameNameNamespaceNoExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "samebloodyname",
			Namespace: "samebloodynamespace",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "f",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm14",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_same_namens_2",
			Namespace: "testing_core_2",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "f",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz14",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name:      strRef("samebloodyname"),
			Namespace: strRef("samebloodynamespace"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "f",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update multiple devices as name/namespace 'samebloodyname/samebloodynamespace' must be unique")
}

func TestDeviceUpdateManyTargetSameNameNamespaceOneExist(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	sb := s.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_same_namens_1",
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "f",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86tlm15",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      "update_dev_same_namens_2",
			Namespace: "testing_core_2",
			Labels: map[string]string{
				"testing": "core",
				"swarm":   "f",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "0xf86xyz15",
		},
	})
	updReq := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name:      strRef("samebloodyname"),
			Namespace: strRef("samebloodynamespace"),
		},
		Targets: &core_apiv1.DeviceTarget{
			Labels: map[string]string{
				"swarm": "f",
			},
		},
	}

	// Act
	_, err := sb.Incubate()
	if err != nil {
		t.Error(err)
	}
	updResp, err := core_client.PublishDeviceUpdateRequest(b, updReq)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, updResp.GetError(), "error updating device: cannot update multiple devices as name/namespace 'samebloodyname/samebloodynamespace' must be unique")
}

func TestDeviceGoesOnline(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_goes_online"}
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: deviceIds,
		},
	}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility": "hvac",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
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
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: deviceIds,
		},
	}
	reqCreate := []*core_apiv1.CreateDeviceRequest{
		{
			Meta: &core_apiv1.Meta{
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
					"utility": "hvac",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: deviceIds[0],
			},
		},
	}

	// Subscribe to device offline event
	offlineEventCount := 0
	s, err := b.Subscribe(
		core_client.DeviceOfflineEvent.WithId(mSdk.GetInstanceName()),
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

	// respListOff, err := core_client.PublishDeviceListRequest(b, reqList)
	// if err != nil {
	// 	t.Error(err)
	// }

	// Assert
	for _, dev := range respListOn.GetOk().Devices {
		switch dev.Spec.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, hbTime.Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
	// TODO this does work, but when running all test, it fails
	// it fails, because the device online goes to another core srv which dies before doing its offline check
	// Solution is to remove the services from test and run them via local or docker
	// for _, dev := range respListOff.GetOk().Devices {
	// 	switch dev.Spec.DeviceId {
	// 	case deviceIds[0]:
	// 		assert.Equal(t, dev.Status.Online, false)
	// 		devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
	// 		assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() > 30, true)
	// 	}
	// }

	// assert.Equal(t, true, offlineEventCount > 0)
	s.Unsubscribe()
}

func TestDeviceAutoProvision(t *testing.T) {
	// Arrange
	deviceIds := []string{"device_auto_provision"}
	reqList := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.DeviceTarget{
			Ids: deviceIds,
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
	createEventCount := 0
	c, err := b.Subscribe(
		core_client.DeviceCreatedEvent.WithId("*"),
		func(msg *nats.Msg) {
			if clients.ServerSubject(msg.Subject).GetId() == deviceIds[0] {
				createEventCount += 1
			}
			msg.Ack()
		})

	// Act
	if err := core_client.PublishHearthbeatStream(b, deviceIds[0]); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respList, err := core_client.PublishDeviceListRequest(b, reqList)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(respList.GetOk().Devices), 1)
	for _, dev := range respList.GetOk().Devices {
		switch dev.Spec.DeviceId {
		case deviceIds[0]:
			assert.Equal(t, dev.Status.Online, true)
			devTs := mir_models.AsGoTime(dev.Status.LastHearthbeat)
			assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() < 10, true)
		}
	}
	assert.Equal(t, 1, onlineEventCount)
	assert.Equal(t, 1, createEventCount)
	s.Unsubscribe()
	c.Unsubscribe()
}

func TestDeviceUpdateDesiredProperties(t *testing.T) {
	// Arrange
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(b)
	id := "update_desired_props"
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}).WithStoreOptions(mirDev.StoreOptions{InMemory: true}).
		WithSchema(core_testv1.File_core_test_v1_core_proto).
		WithConfigHandler(&core_testv1.Conduit{}, func(protoreflect.ProtoMessage) {
			count += 1
		}).
		Incubate(); err != nil {
		t.Error(err)
	}

	prop := &core_testv1.Conduit{
		Power:     5,
		ValveOpen: true,
		GazLevel:  24,
	}
	propName := string(prop.ProtoReflect().Descriptor().FullName())
	st, err := test_utils.ProtoToDesiredStructPb(prop)
	if err != nil {
		t.Error(err)
	}
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: s.ToTarget(),
		Props: &core_apiv1.UpdateDeviceRequest_Properties{
			Desired: st,
		},
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	devs, err := mSdk.Server().UpdateDevice().Request(reqUpd)
	if err != nil {
		t.Error(err)
	}
	dev := devs[0]
	time.Sleep(1 * time.Second)

	// Assert
	assert.Equal(t, dev.Properties.Desired[propName].(map[string]any)["power"], float64(5))
	assert.Equal(t, 2, count)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestDeviceUpdateDesiredPropertiesDoubleSameUpdate(t *testing.T) {
	// Arrange
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(b)
	id := "update_desired_props_double"
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}).WithStoreOptions(mirDev.StoreOptions{InMemory: true}).
		WithSchema(core_testv1.File_core_test_v1_core_proto).
		WithConfigHandler(&core_testv1.Conduit{}, func(protoreflect.ProtoMessage) {
			count += 1
		}).
		Incubate(); err != nil {
		t.Error(err)
	}

	prop := &core_testv1.Conduit{
		Power:     5,
		ValveOpen: true,
		GazLevel:  24,
	}
	propName := string(prop.ProtoReflect().Descriptor().FullName())
	st, err := test_utils.ProtoToDesiredStructPb(prop)
	if err != nil {
		t.Error(err)
	}
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: s.ToTarget(),
		Props: &core_apiv1.UpdateDeviceRequest_Properties{
			Desired: st,
		},
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	devs, err := mSdk.Server().UpdateDevice().Request(reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(2 * time.Second)
	devs, err = mSdk.Server().UpdateDevice().Request(reqUpd)
	if err != nil {
		t.Error(err)
	}
	dev := devs[0]
	time.Sleep(2 * time.Second)

	// Assert
	assert.Equal(t, dev.Properties.Desired[propName].(map[string]any)["power"], float64(5))
	assert.Equal(t, 2, count)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestDeviceUpdateDesiredPropertiesInvalid(t *testing.T) {
	// Arrange
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(b)
	id := "update_desired_props_invalid"
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "core",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}).WithStoreOptions(mirDev.StoreOptions{InMemory: true}).
		WithSchema(core_testv1.File_core_test_v1_core_proto).
		WithConfigHandler(&core_testv1.Conduit{}, func(protoreflect.ProtoMessage) {
			count += 1
		}).
		Incubate(); err != nil {
		t.Error(err)
	}

	prop := &core_testv1.Conduit{
		Power:     5,
		ValveOpen: true,
		GazLevel:  24,
	}
	propName := string(prop.ProtoReflect().Descriptor().FullName())
	propMap, err := test_utils.ProtoToMap(prop)
	propMap["wrong_field"] = "wrong"
	propMap = map[string]any{
		propName: propMap,
	}
	st, err := structpb.NewStruct(propMap)
	if err != nil {
		t.Error(err)
	}
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: s.ToTarget(),
		Props: &core_apiv1.UpdateDeviceRequest_Properties{
			Desired: st,
		},
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	_, err = mSdk.Server().UpdateDevice().Request(reqUpd)

	// Assert
	assert.ErrorContains(t, err, "error validating config")
	assert.Equal(t, 1, count)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestDeviceOnlineEvent(t *testing.T) {
	deviceID := "test-event-online"
	testDevice := mir_models.Device{
		Object: mir_models.Object{
			Meta: mir_models.Meta{
				Name:      "Test_Device",
				Namespace: "default",
			},
		},
		Spec: mir_models.DeviceSpec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := mSdk.Event().DeviceOnline().Subscribe(func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
		received <- device
	})
	if err != nil {
		t.Error(err)
	}

	if err = publishDeviceOnlineEvent(mSdk, nil, testDevice); err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceOfflineEvent(t *testing.T) {
	deviceID := "test-event-offline"
	testDevice := mir_models.Device{
		Object: mir_models.Object{
			Meta: mir_models.Meta{
				Name:      "Test_Device",
				Namespace: "default",
			},
		},
		Spec: mir_models.DeviceSpec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := mSdk.Event().DeviceOffline().Subscribe(func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
		received <- device
	})

	if err = publishDeviceOfflineEvent(mSdk, nil, testDevice); err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceCreatedEvent(t *testing.T) {
	deviceID := "test-event-created"
	testDevice := mir_models.Device{
		Object: mir_models.Object{
			Meta: mir_models.Meta{
				Name:      "Test_Device",
				Namespace: "default",
			},
		},
		Spec: mir_models.DeviceSpec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := mSdk.Event().DeviceCreate().Subscribe(func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
		received <- device
	})

	err = publishDeviceCreateEvent(mSdk, nil, testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceUpdateEvent(t *testing.T) {
	deviceID := "test-event-uodate"
	testDevice := mir_models.Device{
		Object: mir_models.Object{
			Meta: mir_models.Meta{
				Name:      "Test_Device",
				Namespace: "default",
			},
		},
		Spec: mir_models.DeviceSpec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := mSdk.Event().DeviceUpdate().Subscribe(func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
		received <- device
	})

	err = publishDeviceUpdateEvent(mSdk, nil, testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceDeleteEvent(t *testing.T) {
	deviceID := "test-event=delete"
	testDevice := mir_models.Device{
		Object: mir_models.Object{
			Meta: mir_models.Meta{
				Name:      "Test_Device",
				Namespace: "default",
			},
		},
		Spec: mir_models.DeviceSpec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := mSdk.Event().DeviceDelete().Subscribe(func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
		received <- device
	})

	err = publishDeviceDeleteEvent(mSdk, nil, testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func strRef(s string) *string {
	return &s
}
