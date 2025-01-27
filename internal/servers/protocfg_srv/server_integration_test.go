package protocfg_srv

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	protocfg_testv1 "github.com/maxthom/mir/internal/servers/protocfg_srv/proto_test/gen/protocfg_test/v1"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("cfg")
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
	mSdk, err = mir.Connect("test_cfg", busUrl)
	if err != nil {
		panic(err)
	}
	coreSrv, err := core_srv.NewCore(log, mSdk, mng.NewSurrealDeviceStore(db))
	if err := coreSrv.Serve(); err != nil {
		panic(err)
	}
	cfgSrv, err := NewProtoCfg(log, mSdk, mng.NewSurrealDeviceStore(db))
	if err := cfgSrv.Serve(); err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cfg",
	})
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cfg",
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

func TestPublishCfgListRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg"
	s := swarm.NewSwarm(b)
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}).WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).Incubate(); err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	respListCfg, err := cfg_client.PublishListConfigRequest(b, &cfg_apiv1.SendListConfigRequest{
		Targets:       s.ToTarget(),
		FilterLabels:  map[string]string{},
		RefreshSchema: false,
	})
	if err != nil {
		t.Error(err)
	} else if respListCfg.GetError() != "" {
		t.Error(respListCfg.GetError())
	}

	// Assert
	dev := respListCfg.GetOk().DeviceConfigs[id+"/testing_cfg"]
	fmt.Println(dev)
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.Configs), 4)
	assert.Equal(t, dev.Configs[0].Name, "protocfg_test.v1.Conduit")
	assert.Equal(t, dev.Configs[0].Labels["building"], "A")
	assert.Equal(t, dev.Configs[0].Labels["floor"], "1")
	assert.Equal(t, dev.Configs[1].Name, "protocfg_test.v1.PowerLevel")
	assert.Equal(t, dev.Configs[1].Labels["building"], "A")
	assert.Equal(t, dev.Configs[1].Labels["floor"], "2")
	assert.Equal(t, dev.Configs[2].Name, "protocfg_test.v1.Coordinate")
	assert.Equal(t, len(dev.Configs[2].Labels), 0)
	assert.Equal(t, dev.Configs[3].Name, "protocfg_test.v1.Destination")
	assert.Equal(t, len(dev.Configs[3].Labels), 0)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCfgListFiltersRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg_filters"
	s := swarm.NewSwarm(b)
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}).WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).Incubate(); err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	respListCmd, err := cfg_client.PublishListConfigRequest(b, &cfg_apiv1.SendListConfigRequest{
		Targets: s.ToTarget(),
		FilterLabels: map[string]string{
			"building": "A",
			"floor":    "2",
		},
		RefreshSchema: true,
	})
	if err != nil {
		t.Error(err)
	} else if respListCmd.GetError() != "" {
		t.Error(respListCmd.GetError())
	}

	// Assert
	dev := respListCmd.GetOk().DeviceConfigs[id+"/testing_cfg"]
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.Configs), 1)
	assert.Equal(t, dev.Configs[0].Name, "protocfg_test.v1.PowerLevel")
	assert.Equal(t, dev.Configs[0].Labels["building"], "A")
	assert.Equal(t, dev.Configs[0].Labels["floor"], "2")

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCfgRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	props := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		props,
		func(m protoreflect.ProtoMessage) {
			props = m.(*protocfg_testv1.PowerLevel)
		},
	)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		RefreshSchema:   true,
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

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}

	msgResp := &devicev1.Void{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, int32(5), props.Power)
	assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCfgBadRequest(t *testing.T) {
	// Arrange
	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{},
		},
		Name:            "",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_UNSPECIFIED,
		Payload:         []byte{},
	}

	// Act
	respCmd, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, true, strings.Contains(respErr, ""))
}

func TestPublishCfgJsonRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_json"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
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

	respCmd, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	msgResp := &devicev1.Void{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCfgNoDeviceFound(t *testing.T) {
	// Arrange
	id := "no_device_found_cfg"

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, respErr, "error sending config to devices: no device found with current targets criteria")
}

func TestPublishCfgProtoDryRun(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "proto_dryrun_cfg"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		DryRun:          true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respCmd, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED, v.Status)
	}
	cancel()
	wg.Wait()
}

func TestPublishCfgProtoInvalidPayload(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_invalid_payload"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
		},
	)

	reqPayload := protocfg_testv1.Destination{
		Name: "bobby_pendragon",
		Pos: &protocfg_testv1.Coordinate{
			X: 1,
			Y: 2,
			Z: 3,
		},
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            "INVALID",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		RefreshSchema:   true,
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

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	for _, v := range respCfg.GetOk().DeviceResponses {
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, v.Error, "error retrieve config descriptor from device schema: proto: not found")
	}
	cancel()
	wg.Wait()
}

func TestPublishCfgRequestMultipleDevices(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "device_send_cfg_1",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "device_send_cfg_2",
		},
	}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets:         swarm.ToTarget(),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &devicev1.Void{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, int32(5), cfgHandled.Power)
		assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
	}

	assert.Equal(t, len(swarm.Devices), handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesOneNoHandler(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "device_send_cfg_1_no_handler",
		},
	}, &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "device_send_cfg_2_no_handler",
		},
	}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	swarm.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: "device_send_cfg_3_no_handler",
		},
	}).WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &cfg_apiv1.SendConfigRequest{
		Targets:         swarm.ToTarget(),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &devicev1.Void{}
	for k, v := range respCfg.GetOk().DeviceResponses {
		if k == "device_send_cfg_1/testing_cfg" || k == "device_send_cfg_2/testing_cfg" {
			if v.Error != "" {
				t.Error(v.Error)
			}
			assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
			assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, int32(5), cfgHandled.Power)
			assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
		} else if k == "device_send_cmd_3/testing_cfg" {
			assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, v.Status)
			assert.Equal(t, "device error: no handler for command protocmd_test.v1.PowerLevel found", v.Error)
		}
	}

	assert.Equal(t, 2, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesJson(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_send_cfg_multi_json_1",
			},
		}, &core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_send_cfg_multi_json_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, payloadName, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets:         swarm.ToTarget(),
		Name:            payloadName,
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &devicev1.Void{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, reqPayload.Power, cfgHandled.Power)
		assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	}

	assert.Equal(t, len(swarm.Devices), handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesDescriptorNotFound(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_cfg_send_notfound_1",
			},
		}, &core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_cfg_send_notfound_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, _, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets:         swarm.ToTarget(),
		Name:            "nothing",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	// msgResp := &protocfg_testv1.PowerLevelReport{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		// assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, true, v.Error != "")
	}

	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, 0, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesSingleDescriptorNotFoundForcePush(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevice(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_valid_cfg",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	_, err = swarm.AddDevice(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_invalid_cfg",
			},
		}).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, payloadName, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets:         swarm.ToTarget(),
		Name:            payloadName,
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		ForcePush:       true,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	devValid := respCfg.GetOk().DeviceResponses["device_valid_cfg/testing_cfg"]
	devInvalid := respCfg.GetOk().DeviceResponses["device_invalid_cfg/testing_cfg"]

	msgResp := &devicev1.Void{}
	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, devValid.Status)
	if err = protojson.Unmarshal(devValid.Payload, msgResp); err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), devValid.Name)
	assert.Equal(t, reqPayload.Power, cfgHandled.Power)

	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, devInvalid.Status)
	assert.Equal(t, true, devInvalid.Error != "")

	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, 1, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesJsonTemplate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	swarm := swarm.NewSwarm(b)
	_, err := swarm.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_send_cfg_multi_json_tlm_1",
			},
		}, &core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "device_send_cfg_multi_json_tlm_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{}
	cmdName := string(reqPayload.ProtoReflect().Descriptor().FullName())
	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets:      swarm.ToTarget(),
		Name:         cmdName,
		ShowTemplate: true,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	resp := respCfg.GetOk()
	dev1 := resp.DeviceResponses["device_send_cfg_multi_json_tlm_1/testing_cfg"]
	dev2 := resp.DeviceResponses["device_send_cfg_multi_json_tlm_2/testing_cfg"]
	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, cmdName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev2.Status)
	assert.Equal(t, cmdName, dev2.Name)
	assert.Equal(t, `{"power":0}`, string(dev2.Payload))

	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesOneTimeoutJsonTemplate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_multi_timeout_json_template_1"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	id2 := "device_send_cfg_multi_timeout_json_template_2"
	reqCreate2 := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id2,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id2,
		},
	}

	reqCfgPayload := &protocfg_testv1.PowerLevel{}
	reqCfgName := string(reqCfgPayload.ProtoReflect().Descriptor().FullName())
	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id, id2},
		},
		Name:         reqCfgName,
		ShowTemplate: true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	resp := respCmd.GetOk()
	dev1 := resp.DeviceResponses[id+"/testing_cfg"]
	dev2 := resp.DeviceResponses[id2+"/testing_cfg"]
	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, reqCfgName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, dev2.Status)
	assert.Equal(t, "", dev2.Name)
	assert.Equal(t, "", string(dev2.Payload))
	assert.Equal(t, dev2.Error, "error retrieve config descriptor from device schema: cannot reconcile device schema: error requesting device schema: error publishing request message: nats: no responders available for request")

	cancel()
	wg.Wait()
}

func TestPublishCfgJsonNameWithCurlyRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_json_curly"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &cfg_apiv1.SendConfigRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()) + "{x=y}",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
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

	respCfg, err := cfg_client.PublishSendConfigRequest(b, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	msgResp := &devicev1.Void{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	cancel()
	wg.Wait()
}
