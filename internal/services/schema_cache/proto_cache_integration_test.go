package schema_cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	schemacache_testv1 "github.com/maxthom/mir/internal/services/schema_cache/proto_test/gen/schemacache_test/v1"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_models"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
)

var db *surrealdb.DB
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var logTest = logger.With().Str("test", "schema_cache").Logger().Level(zerolog.DebugLevel)

var b *bus.BusConn

func TestMain(m *testing.M) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println("Test Setup")

	b, db, _, _, _ = test_utils.SetupAllExternalsPanic(ctx, test_utils.ConnsInfo{
		Name:   "test_prototlm",
		BusUrl: busUrl,
		Surreal: test_utils.SurrealInfo{
			Url:  "ws://127.0.0.1:8000/rpc",
			User: "root",
			Pass: "root",
			Ns:   "global",
			Db:   "mir_testing",
		},
		Influx: test_utils.InfluxInfo{
			Url:    "http://localhost:8086/",
			Token:  "mir-operator-token",
			Org:    "Mir",
			Bucket: "mir_integration_test",
		},
	})
	var err error
	mSdk, err = mir.Connect("test_protocache", busUrl)
	if err != nil {
		panic(err)
	}
	coreSrv, err := core_srv.NewCore(logTest, mSdk, mng.NewSurrealDeviceStore(db))
	if err != nil {
		panic(err)
	}
	if err = coreSrv.Serve(); err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "proto_cache",
	})
	time.Sleep(1 * time.Second)
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "proto_cache",
	})
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	cancel()
	b.Close()
	mSdk.Disconnect()
	db.Close()
	fmt.Println(" -> closed connections")

	os.Exit(exitVal)
}

func TestPublishDeviceUpdateCache(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())

	cache, err := NewMirProtoCache(l, mSdk)
	if err != nil {
		t.Error(err)
	}
	id := "device_proto_cache"
	count := 0
	cache.AddDeviceUpdateSub(func(deviceId string, device mir_models.Device, schema mir_proto.MirProtoSchema) {
		if deviceId == id {
			count++
		}
	})

	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "proto_cache",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		schemacache_testv1.File_schemacache_test_v1_cache_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	sch, _, err := cache.GetDeviceSchema(id, false)
	ogSch, _ := mir_proto.NewMirProtoSchema(
		schemacache_testv1.File_schemacache_test_v1_cache_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)

	str := "update"
	if _, err = core_client.PublishDeviceUpdateRequest(b, &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
				"test": {
					Value: &str,
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)
	_, devPostUpd, err := cache.GetDeviceSchema(id, false)

	// Assert
	assert.Equal(t, true, mir_proto.AreSchemaEqual(ogSch, sch))
	assert.Equal(t, devPostUpd.Meta.Labels["test"], str)
	assert.Equal(t, 2, count)
	cancel()
	wg.Wait()
}
