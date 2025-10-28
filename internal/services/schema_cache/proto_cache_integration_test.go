package schema_cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/test_utils"
	schemacache_testv1 "github.com/maxthom/mir/internal/services/schema_cache/proto_test/gen/schemacache_test/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/module/mir"
)

var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var log = test_utils.TestLogger("schema_cache")

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	var err error
	mSdk, err = mir.Connect(log, "test_protocache", busUrl)
	if err != nil {
		panic(err)
	}
	if err := dataCleanUp(); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)

	// Tests
	fmt.Println("> Test Run")
	exitVal := m.Run()
	time.Sleep(1 * time.Second)

	// Teardown
	fmt.Println("> Test Teardown")
	if err := mSdk.Disconnect(); err != nil {
		fmt.Println(err)
	}
	time.Sleep(1 * time.Second)

	os.Exit(exitVal)
}

func dataCleanUp() error {
	if _, err := mSdk.Client().DeleteDevice().Request(mir_v1.DeviceTarget{
		Labels: map[string]string{
			"testing": "proto_cache",
		},
	}); err != nil {
		return err
	}
	return nil
}

func TestPublishDeviceUpdateCache(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())

	cache, err := NewMirSchemaCache(log, mSdk)
	if err != nil {
		t.Error(err)
	}
	id := "device_proto_cache"
	count := 0
	cache.AddDeviceUpdateSub(func(deviceId string, device mir_v1.Device, schema mir_proto.MirProtoSchema) {
		if deviceId == id {
			count++
		}
	})

	reqCreate := &mir_apiv1.CreateDeviceRequest{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "proto_cache",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().ExcludeSchemaOnLaunch().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		schemacache_testv1.File_schemacache_test_v1_cache_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus, reqCreate)
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
	if _, err = core_client.PublishDeviceUpdateRequest(mSdk.Bus, &mir_apiv1.UpdateDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Meta: &mir_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*mir_apiv1.OptString{
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
	assert.Equal(t, 1, count)
	cancel()
	wg.Wait()
}
