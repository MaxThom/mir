package protocfg_srv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	protocfg_testv1 "github.com/maxthom/mir/internal/servers/protocfg_srv/proto_test/gen/protocfg_test/v1"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("cfg")
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	var err error

	mSdk, err = mir.Connect("test_cfg", busUrl)
	if err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
	if err := dataCleanUp(); err != nil {
		panic(err)
	}

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
	// Clear data
	if _, err := mSdk.Client().DeleteDevice().Request(mir_v1.DeviceTarget{
		Labels: map[string]string{
			"testing": "cfg",
		},
	}); err != nil {
		return err
	}
	return nil
}

func TestPublishCfgListRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg"
	s := swarm.NewSwarm(mSdk.Bus)
	if _, err := s.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
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

	respListCfg, err := cfg_client.PublishListConfigRequest(mSdk.Bus, &mir_apiv1.SendListConfigRequest{
		Targets:       mir_v1.MirDeviceTargetToProtoDeviceTarget(s.ToTarget()),
		FilterLabels:  map[string]string{},
		RefreshSchema: false,
	})
	if err != nil {
		t.Error(err)
	} else if respListCfg.GetError() != "" {
		t.Error(respListCfg.GetError())
	}

	// Assert
	dev := respListCfg.GetOk().DeviceConfigs[0].CfgDescriptors
	assert.Equal(t, dev[0].Error, "")
	assert.Equal(t, len(dev), 4)
	assert.Equal(t, dev[0].Name, "protocfg_test.v1.Conduit")
	assert.Equal(t, dev[0].Labels["building"], "A")
	assert.Equal(t, dev[0].Labels["floor"], "1")
	assert.Equal(t, dev[1].Name, "protocfg_test.v1.PowerLevel")
	assert.Equal(t, dev[1].Labels["building"], "A")
	assert.Equal(t, dev[1].Labels["floor"], "2")
	assert.Equal(t, dev[2].Name, "protocfg_test.v1.Coordinate")
	assert.Equal(t, len(dev[2].Labels), 0)
	assert.Equal(t, dev[3].Name, "protocfg_test.v1.Destination")
	assert.Equal(t, len(dev[3].Labels), 0)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCfgListRequestWithValues(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg_values"
	s := swarm.NewSwarm(mSdk.Bus)
	if _, err := s.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}).WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).Incubate(); err != nil {
		t.Error(err)
	}
	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	time.Sleep(1 * time.Second)
	respListCfg, err := cfg_client.PublishListConfigRequest(mSdk.Bus, &mir_apiv1.SendListConfigRequest{
		Targets:       mir_v1.MirDeviceTargetToProtoDeviceTarget(s.ToTarget()),
		FilterLabels:  map[string]string{},
		RefreshSchema: false,
	})
	if err != nil {
		t.Error(err)
	} else if respListCfg.GetError() != "" {
		t.Error(respListCfg.GetError())
	}

	// Assert
	cfgDesc := respListCfg.GetOk().DeviceConfigs[0].CfgDescriptors
	assert.Equal(t, cfgDesc[0].Error, "")
	assert.Equal(t, len(cfgDesc), 4)
	assert.Equal(t, cfgDesc[0].Name, "protocfg_test.v1.Conduit")
	assert.Equal(t, cfgDesc[0].Labels["building"], "A")
	assert.Equal(t, cfgDesc[0].Labels["floor"], "1")
	assert.Equal(t, cfgDesc[1].Name, "protocfg_test.v1.PowerLevel")
	assert.Equal(t, cfgDesc[1].Labels["building"], "A")
	assert.Equal(t, cfgDesc[1].Labels["floor"], "2")
	assert.Equal(t, cfgDesc[2].Name, "protocfg_test.v1.Coordinate")
	assert.Equal(t, len(cfgDesc[2].Labels), 0)
	assert.Equal(t, cfgDesc[3].Name, "protocfg_test.v1.Destination")
	assert.Equal(t, len(cfgDesc[3].Labels), 0)

	cfgValues := respListCfg.GetOk().DeviceConfigs[0].CfgValues
	assert.Equal(t, len(cfgValues), 1)
	assert.Equal(t, cfgValues[0].Id.DeviceId, id)
	assert.Equal(t, cfgValues[0].Id.Name, id)
	assert.Equal(t, cfgValues[0].Id.Namespace, "testing_cfg")
	assert.Equal(t, len(cfgValues[0].Values), 4)

	val, ok := cfgValues[0].Values["protocfg_test.v1.PowerLevel"]
	assert.Equal(t, ok, true)
	assert.Equal(t, val, "{\"power\":5}")
	val, ok = cfgValues[0].Values["protocfg_test.v1.Conduit"]
	assert.Equal(t, ok, true)
	assert.Equal(t, val, "{\"gazLevel\":0,\"power\":0,\"valveOpen\":false}")
	val, ok = cfgValues[0].Values["protocfg_test.v1.Coordinate"]
	assert.Equal(t, ok, true)
	assert.Equal(t, val, "{\"x\":0,\"y\":0,\"z\":0}")
	val, ok = cfgValues[0].Values["protocfg_test.v1.Destination"]
	assert.Equal(t, ok, true)
	assert.Equal(t, val, "{\"name\":\"\",\"pos\":{\"x\":0,\"y\":0,\"z\":0}}")

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}
func TestPublishCfgListFiltersRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg_filters"
	s := swarm.NewSwarm(mSdk.Bus)
	if _, err := s.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
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

	respListCmd, err := cfg_client.PublishListConfigRequest(mSdk.Bus, &mir_apiv1.SendListConfigRequest{
		Targets: mir_v1.MirDeviceTargetToProtoDeviceTarget(s.ToTarget()),
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
	fmt.Println(respListCmd)
	dev := respListCmd.GetOk().DeviceConfigs[0]
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.CfgDescriptors), 1)
	assert.Equal(t, dev.CfgDescriptors[0].Name, "protocfg_test.v1.PowerLevel")
	assert.Equal(t, dev.CfgDescriptors[0].Labels["building"], "A")
	assert.Equal(t, dev.CfgDescriptors[0].Labels["floor"], "2")

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCfgRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	props := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		props,
		func(m protoreflect.ProtoMessage) {
			props = m.(*protocfg_testv1.PowerLevel)
			chHdlr <- struct{}{}
		},
	)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}

	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	// Assert
	assert.Equal(t, int32(5), props.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCfgBadRequest(t *testing.T) {
	// Arrange
	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{},
		},
		Name:            "",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_UNSPECIFIED,
		Payload:         []byte{},
	}

	// Act
	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
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
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	// Assert
	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCfgCurrentValues(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_current_values"
	nameNs := id + "/testing_cfg"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	reqCurrent := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		ShowValues:      true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	respCurrent, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCurrent)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	devVal, ok := respCurrent.GetOk().DeviceResponses[nameNs]
	assert.Equal(t, true, ok)
	assert.Equal(t, devVal.Status, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS)
	assert.Equal(t, devVal.DeviceId, id)
	assert.Equal(t, devVal.Name, string(msgResp.ProtoReflect().Descriptor().FullName()))
	assert.Equal(t, devVal.Error, "")

	if err = json.Unmarshal(devVal.Payload, msgResp); err != nil {
		t.Error(err)
	}
	assert.Equal(t, msgResp.Power, int32(5))

	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCfgRequestCheckTime(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_time"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	reqDev := &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	respUpd, err := core_client.PublishDeviceListRequest(mSdk.Bus, reqDev)
	if err != nil {
		t.Error(err)
	}
	if respUpd.GetError() != "" {
		t.Error(respUpd.GetError())
	}

	// Assert
	cfgTime := respUpd.GetOk().GetDevices()[0].Status.Properties.Desired[string(p.ProtoReflect().Descriptor().FullName())]
	assert.Equal(t, true, time.Until(mir_v1.AsGoTime(cfgTime)) < time.Second*10)

	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
	time.Sleep(1 * time.Second)
}

func TestPublishCfgDoubleUpdateSendIfDifferent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_json"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	count := 0
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			count += 1
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytes,
		RefreshSchema:     true,
		SendOnlyDifferent: false,
	}

	reqCfgSec := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytes,
		SendOnlyDifferent: true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	respCfgSec, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfgSec)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_NOCHANGE, respCfgSec.GetOk().DeviceResponses[id+"/testing_cfg"].Status)
	assert.Equal(t, 2, count)
	cancel()
	wg.Wait()
}

func TestPublishCfgDoubleUpdateSendAlways(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_json"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	count := 0
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			count += 1
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytes,
		RefreshSchema:     true,
		SendOnlyDifferent: false,
	}

	reqCfgSec := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytes,
		SendOnlyDifferent: false,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	respCfgSec, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfgSec)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, respCfgSec.GetOk().DeviceResponses[id+"/testing_cfg"].Status)
	assert.Equal(t, 3, count)
	cancel()
	wg.Wait()
}

func TestPublishCfgDoubleUpdateIsDifferent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_json"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	count := 0
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			count += 1
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytes,
		RefreshSchema:     true,
		SendOnlyDifferent: false,
	}

	pSec := protocfg_testv1.PowerLevel{
		Power: 10,
	}
	payloadBytesSec, err := protojson.Marshal(&pSec)
	if err != nil {
		t.Error(err)
	}

	reqCfgSec := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:              string(p.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding:   mir_apiv1.Encoding_ENCODING_JSON,
		Payload:           payloadBytesSec,
		SendOnlyDifferent: true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	respCfgSec, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfgSec)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(10), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, respCfgSec.GetOk().DeviceResponses[id+"/testing_cfg"].Status)
	assert.Equal(t, 3, count)
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
	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, respErr, "error sending config to devices: no device found with current targets criteria")
}

func TestPublishCfgListNoDeviceFound(t *testing.T) {
	// Arrange — no device creation; target ID does not exist in DB

	// Act
	resp, err := cfg_client.PublishListConfigRequest(mSdk.Bus, &mir_apiv1.SendListConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{"nonexistent_cfg_device"},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, "no device found with current targets criteria", resp.GetError())
}

func TestPublishCfgProtoDryRun(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "proto_dryrun_cfg"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		DryRun:          true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
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
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED, v.Status)
	}
	cancel()
	wg.Wait()
}

func TestPublishCfgProtoInvalidPayload(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_invalid_payload"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
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

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            "INVALID",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	for _, v := range respCfg.GetOk().DeviceResponses {
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, strings.Contains(v.Error, "error retrieve config descriptor from device schema"), true)
	}
	cancel()
	wg.Wait()
}

func TestPublishCfgRequestMultipleDevices(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	chHdlr := make(chan struct{}, 4)
	_, err := swarm.AddDevices(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cfg_1",
		},
	}, &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cfg_2",
		},
	}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
				chHdlr <- struct{}{}
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
	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	for i := 0; i < len(swarm.Devices)*2; i++ {
		<-chHdlr
	}

	assert.Equal(t, len(swarm.Devices)*2, handlerCount)
	assert.Equal(t, int32(5), cfgHandled.Power)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesOneNoHandler(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	chHdlr := make(chan struct{}, 4)
	if _, err := swarm.AddDevices(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cfg_1_no_handler",
		},
	}, &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cfg_2_no_handler",
		},
	}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
				chHdlr <- struct{}{}
			},
		).Incubate(); err != nil {
		t.Error(err)
	}
	if _, err := swarm.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cfg_3_no_handler",
		},
	}).WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).Incubate(); err != nil {
		t.Error(err)
	}
	wg, err := swarm.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for k, v := range respCfg.GetOk().DeviceResponses {
		if k == "device_send_cfg_1_no_handler/testing_cfg" || k == "device_send_cfg_2_no_handler/testing_cfg" {
			if v.Error != "" {
				t.Error(v.Error)
			}
			assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
			assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
		} else if k == "device_send_cfg_3_no_handler/testing_cfg" {
			assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	for i := 0; i < 4; i++ {
		<-chHdlr
	}
	assert.Equal(t, 4, handlerCount)
	assert.Equal(t, int32(5), cfgHandled.Power)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesJson(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	chHdlr := make(chan struct{}, 4)
	_, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cfg_multi_json_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cfg_multi_json_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
				chHdlr <- struct{}{}
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
	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            payloadName,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	for i := 0; i < 4; i++ {
		<-chHdlr
	}

	assert.Equal(t, reqPayload.Power, cfgHandled.Power)
	assert.Equal(t, len(swarm.Devices)*2, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesDescriptorNotFound(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	chHdlr := make(chan struct{}, 2)
	_, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_cfg_send_notfound_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_cfg_send_notfound_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
				chHdlr <- struct{}{}
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
	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            "nothing",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	for _, v := range respCfg.GetOk().DeviceResponses {
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, true, v.Error != "")
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, 2, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesSingleDescriptorNotFoundForcePush(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cfgHandled := &protocfg_testv1.PowerLevel{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	chHdlr := make(chan struct{}, 2)
	_, err := swarm.AddDevice(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_valid_cfg",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		WithConfigHandler(
			cfgHandled,
			func(m protoreflect.ProtoMessage) {
				cfgHandled = m.(*protocfg_testv1.PowerLevel)
				handlerCount++
				chHdlr <- struct{}{}
			},
		).Incubate()
	_, err = swarm.AddDevice(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
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
	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            payloadName,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		ForcePush:       true,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	devValid := respCfg.GetOk().DeviceResponses["device_valid_cfg/testing_cfg"]
	devInvalid := respCfg.GetOk().DeviceResponses["device_invalid_cfg/testing_cfg"]

	msgResp := &protocfg_testv1.PowerLevel{}
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, devValid.Status)
	if err = protojson.Unmarshal(devValid.Payload, msgResp); err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), devValid.Name)
	assert.Equal(t, reqPayload.Power, cfgHandled.Power)

	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, devInvalid.Status)
	assert.Equal(t, true, devInvalid.Error != "")

	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	assert.Equal(t, 2, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCfgRequestMultipleDevicesJsonTemplate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	swarm := swarm.NewSwarm(mSdk.Bus)
	respD, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cfg_multi_json_tlm_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cfg",
				Labels: map[string]string{
					"testing": "cfg",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cfg_multi_json_tlm_2",
			},
		}).
		WithSchema(protocfg_testv1.File_protocfg_test_v1_cfg_proto).
		Incubate()
	fmt.Println(respD)
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocfg_testv1.PowerLevel{}
	cmdName := string(reqPayload.ProtoReflect().Descriptor().FullName())
	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets:      mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:         cmdName,
		ShowTemplate: true,
	}

	// Act
	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	// Assert
	resp := respCfg.GetOk()
	dev1 := resp.DeviceResponses["device_send_cfg_multi_json_tlm_1/testing_cfg"]
	dev2 := resp.DeviceResponses["device_send_cfg_multi_json_tlm_2/testing_cfg"]
	fmt.Println(resp)
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, cmdName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev2.Status)
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
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	id2 := "device_send_cfg_multi_timeout_json_template_2"
	reqCreate2 := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id2,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id2,
		},
	}

	reqCfgPayload := &protocfg_testv1.PowerLevel{}
	reqCfgName := string(reqCfgPayload.ProtoReflect().Descriptor().FullName())
	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id, id2},
		},
		Name:         reqCfgName,
		ShowTemplate: true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate2,
		})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	resp := respCmd.GetOk()
	dev1 := resp.DeviceResponses[id+"/testing_cfg"]
	dev2 := resp.DeviceResponses[id2+"/testing_cfg"]
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, reqCfgName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR, dev2.Status)
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
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	chHdlr := make(chan struct{}, 2)
	cfgHandled := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		cfgHandled,
		func(m protoreflect.ProtoMessage) {
			cfgHandled = m.(*protocfg_testv1.PowerLevel)
			chHdlr <- struct{}{}
		})

	p := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCfg := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(p.ProtoReflect().Descriptor().FullName()) + "{x=y}",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCfg)
	if err != nil {
		t.Error(err)
	} else if respCfg.GetError() != "" {
		t.Error(respCfg.GetError())
	}

	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// We wait twice here as handler is called twice (one on init, one on update)
	<-chHdlr
	<-chHdlr

	// Assert
	assert.Equal(t, int32(5), cfgHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCfg.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishReportedProperties(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_reported_props"

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(7 * time.Second)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	if err = dev.SendProperties(&reqPayload); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	listResp, err := core_client.PublishDeviceListRequest(mSdk.Bus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{Ids: []string{id}},
	})
	if err != nil {
		t.Error(err)
	} else if listResp.GetError() != "" {
		t.Error(listResp.GetError())
	}

	d := mir_v1.NewDeviceFromProtoDevice(listResp.GetOk().Devices[0])
	// Assert
	assert.Equal(t, float64(5), d.Properties.Reported["protocfg_test.v1.PowerLevel"].(map[string]any)["power"].(float64))
	cancel()
	wg.Wait()
}

func TestPublishReportedPropertiesEvent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_reported_props_event"

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	eventCount := 0
	chHdlr := make(chan struct{}, 1)
	if err = mSdk.Event().ReportedProperties().Subscribe(
		func(msg *mir.Msg, deviceId string, props map[string]any, err error) {
			if deviceId == id {
				eventCount += 1
				chHdlr <- struct{}{}
			}
		}); err != nil {
		t.Error(err)
	}

	// Act
	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(10 * time.Second)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	if err = dev.SendProperties(&reqPayload); err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	listResp, err := core_client.PublishDeviceListRequest(mSdk.Bus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{Ids: []string{id}},
	})
	if err != nil {
		t.Error(err)
	} else if listResp.GetError() != "" {
		t.Error(listResp.GetError())
	}

	d := mir_v1.NewDeviceFromProtoDevice(listResp.GetOk().Devices[0])
	// Assert
	<-chHdlr
	assert.Equal(t, float64(5), d.Properties.Reported["protocfg_test.v1.PowerLevel"].(map[string]any)["power"].(float64))
	assert.Equal(t, 1, eventCount)
	cancel()
	wg.Wait()
}

func TestPublishDesiredPropertiesEvent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cfg_event"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	eventCount := 0
	chHdlr := make(chan struct{}, 3)
	if err := mSdk.Event().DesiredProperties().Subscribe(
		func(msg *mir.Msg, deviceId string, props map[string]any, err error) {
			if deviceId == id {
				eventCount += 1
				chHdlr <- struct{}{}
			}
		}); err != nil {
		t.Error(err)
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
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
			chHdlr <- struct{}{}
		},
	)

	reqPayload := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		RefreshSchema:   true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCfg, err := cfg_client.PublishSendConfigRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}

	msgResp := &protocfg_testv1.PowerLevel{}
	for _, v := range respCfg.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	<-chHdlr
	<-chHdlr
	<-chHdlr

	assert.Equal(t, int32(5), props.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCfg.GetOk().Encoding)
	assert.Equal(t, 1, eventCount)
	cancel()
	wg.Wait()
}

func TestDesiredPropertiesDefaultWritten(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_request_desired_props_default"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	reqGet := &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// This should be called instantly after launch
	propsPowerLevel := &protocfg_testv1.PowerLevel{}
	count := 0
	dev.HandleProperties(
		propsPowerLevel,
		func(m protoreflect.ProtoMessage) {
			propsPowerLevel = m.(*protocfg_testv1.PowerLevel)
			count += 1
		},
	)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	// This should be called after launch with reconciled properties
	chHdlr := make(chan struct{}, 1)
	propsConduit := &protocfg_testv1.Conduit{}
	dev.HandleProperties(
		propsConduit,
		func(m protoreflect.ProtoMessage) {
			propsConduit = m.(*protocfg_testv1.Conduit)
			count += 1
			chHdlr <- struct{}{}
		},
	)
	<-chHdlr

	devs, err := core_client.PublishDeviceListRequest(mSdk.Bus, reqGet)
	if err != nil {
		t.Error(err)
	}
	if devs.GetError() != "" {
		t.Error(devs.GetError())
	}
	d := devs.GetOk().Devices[0].GetProperties().GetDesired()
	m := d.AsMap()

	// Assert
	_, ok := m["protocfg_test.v1.PowerLevel"]
	assert.Equal(t, true, ok)
	_, ok = m["protocfg_test.v1.Conduit"]
	assert.Equal(t, true, ok)
	_, ok = m["protocfg_test.v1.Coordinate"]
	assert.Equal(t, true, ok)
	_, ok = m["protocfg_test.v1.Destination"]
	assert.Equal(t, true, ok)

	assert.Equal(t, int32(0), propsPowerLevel.Power)
	assert.Equal(t, false, propsConduit.ValveOpen)
	assert.Equal(t, 2, count)
	cancel()
	wg.Wait()
}

// The properties are written to db before the device is launched
// We test if the device reconcile its properties and call the handlers
func TestDesiredPropertiesRequestFromDeviceOnBoot(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_request_desired_props"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cfg",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	sch, err := mir_proto.NewMirProtoSchema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	if err != nil {
		t.Error(err)
	}
	compSch, err := sch.CompressSchema()
	if err != nil {
		t.Error(err)
	}
	packNames := sch.GetPackageList()

	updateSchReq := &mir_apiv1.UpdateDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Status: &mir_apiv1.UpdateDeviceRequest_Status{
			Schema: &mir_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: compSch,
				PackageNames:     packNames,
			},
		},
	}

	reqPayloadPowerLevel := protocfg_testv1.PowerLevel{
		Power: 5,
	}
	payloadBytesPowerLevel, err := proto.Marshal(&reqPayloadPowerLevel)
	if err != nil {
		t.Error(err)
	}

	reqPowerLevel := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayloadPowerLevel.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytesPowerLevel,
	}

	reqPayloadConduit := protocfg_testv1.Conduit{
		Power:     5,
		ValveOpen: true,
	}
	payloadBytesConduit, err := proto.Marshal(&reqPayloadConduit)
	if err != nil {
		t.Error(err)
	}

	reqConduit := &mir_apiv1.SendConfigRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayloadConduit.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytesConduit,
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		protocfg_testv1.File_protocfg_test_v1_cfg_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	resp, err := core_client.PublishDeviceUpdateRequest(mSdk.Bus, updateSchReq)
	if err != nil {
		t.Error(err)
	}
	if resp.GetError() != "" {
		t.Error(resp.GetError())
	}
	time.Sleep(1 * time.Second)

	_, err = cfg_client.PublishSendConfigRequest(mSdk.Bus, reqPowerLevel)
	if err != nil {
		t.Error(err)
	}
	_, err = cfg_client.PublishSendConfigRequest(mSdk.Bus, reqConduit)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	// This should be called instantly after launch
	propsPowerLevel := &protocfg_testv1.PowerLevel{}
	dev.HandleProperties(
		propsPowerLevel,
		func(m protoreflect.ProtoMessage) {
			propsPowerLevel = m.(*protocfg_testv1.PowerLevel)
		},
	)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	// This should be called after launch with reconciled properties
	propsConduit := &protocfg_testv1.Conduit{}
	dev.HandleProperties(
		propsConduit,
		func(m protoreflect.ProtoMessage) {
			propsConduit = m.(*protocfg_testv1.Conduit)
		},
	)
	time.Sleep(10 * time.Second)

	// Assert
	assert.Equal(t, int32(5), propsPowerLevel.Power)
	assert.Equal(t, true, propsConduit.ValveOpen)
	cancel()
	wg.Wait()
}
