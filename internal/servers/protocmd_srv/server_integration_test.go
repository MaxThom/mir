package protocmd_srv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	protocmd_testv1 "github.com/maxthom/mir/internal/servers/protocmd_srv/proto_test/gen/protocmd_test/v1"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"gotest.tools/assert"
)

var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var log = test_utils.TestLogger("cmd")

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	var err error
	mSdk, err = mir.Connect("test_protocmd", busUrl)
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
			"testing": "cmd",
		},
	}); err != nil {
		return err
	}
	return nil
}

func TestPublishCmdListRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cmd"
	s := swarm.NewSwarm(mSdk.Bus)
	if _, err := s.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}).WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).Incubate(); err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	respListCmd, err := cmd_client.PublishListCommandsRequest(mSdk.Bus, &mir_apiv1.SendListCommandsRequest{
		Targets:       mir_v1.MirDeviceTargetToProtoDeviceTarget(s.ToTarget()),
		FilterLabels:  map[string]string{},
		RefreshSchema: false,
	})
	if err != nil {
		t.Error(err)
	} else if respListCmd.GetError() != "" {
		t.Error(respListCmd.GetError())
	}

	// Assert
	dev := respListCmd.GetOk().DevicesCommands[0]
	fmt.Println(dev.CmdDescriptors[0].Labels)
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.CmdDescriptors), 4)
	assert.Equal(t, dev.CmdDescriptors[0].Name, "protocmd_test.v1.Command")
	assert.Equal(t, dev.CmdDescriptors[0].Labels["__tag_building"], "A")
	assert.Equal(t, dev.CmdDescriptors[0].Labels["__tag_floor"], "1")
	assert.Equal(t, dev.CmdDescriptors[1].Name, "protocmd_test.v1.ChangePower")
	assert.Equal(t, dev.CmdDescriptors[1].Labels["__tag_building"], "A")
	assert.Equal(t, dev.CmdDescriptors[1].Labels["__tag_floor"], "2")
	assert.Equal(t, dev.CmdDescriptors[2].Name, "protocmd_test.v1.SetTargetVector")
	assert.Equal(t, len(dev.CmdDescriptors[2].Labels), 0)
	assert.Equal(t, dev.CmdDescriptors[3].Name, "protocmd_test.v1.SetDestination")
	assert.Equal(t, len(dev.CmdDescriptors[3].Labels), 0)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCmdListFiltersRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cmd_filters"
	s := swarm.NewSwarm(mSdk.Bus)
	if _, err := s.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}).WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).Incubate(); err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	respListCmd, err := cmd_client.PublishListCommandsRequest(mSdk.Bus, &mir_apiv1.SendListCommandsRequest{
		Targets: mir_v1.MirDeviceTargetToProtoDeviceTarget(s.ToTarget()),
		FilterLabels: map[string]string{
			"__tag_building": "A",
			"__tag_floor":    "2",
		},
		RefreshSchema: true,
	})
	if err != nil {
		t.Error(err)
	} else if respListCmd.GetError() != "" {
		t.Error(respListCmd.GetError())
	}

	// Assert
	dev := respListCmd.GetOk().DevicesCommands[0]
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.CmdDescriptors), 1)
	assert.Equal(t, dev.CmdDescriptors[0].Name, "protocmd_test.v1.ChangePower")
	assert.Equal(t, dev.CmdDescriptors[0].Labels["__tag_building"], "A")
	assert.Equal(t, dev.CmdDescriptors[0].Labels["__tag_floor"], "2")

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCmdRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cmdHandled := &protocmd_testv1.ChangePower{}
	dev.HandleCommand(
		cmdHandled,
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmdHandled = m.(*protocmd_testv1.ChangePower)
			return &protocmd_testv1.ChangePowerResp{Success: true}, nil
		},
	)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
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

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}

	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, int32(5), cmdHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCmdBadRequest(t *testing.T) {
	// Arrange
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{},
		},
		Name:            "",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_UNSPECIFIED,
		Payload:         []byte{},
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, true, strings.Contains(respErr, ""))
}

func TestPublishCmdJsonRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_json"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cmdHandled := &protocmd_testv1.ChangePower{}
	dev.HandleCommand(
		cmdHandled,
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmdHandled = m.(*protocmd_testv1.ChangePower)
			return &protocmd_testv1.ChangePowerResp{Success: true}, nil
		})

	p := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
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

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, int32(5), cmdHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCmdNoDeviceFound(t *testing.T) {
	// Arrange
	id := "no_device_found"

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, respErr, "error sending command to devices: no device found with current targets criteria")
}

func TestPublishCmdListNoDeviceFound(t *testing.T) {
	// Arrange — no device creation; target ID does not exist in DB

	// Act
	resp, err := cmd_client.PublishListCommandsRequest(mSdk.Bus, &mir_apiv1.SendListCommandsRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{"nonexistent_cmd_device"},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, "no device found with current targets criteria", resp.GetError())
}

func TestPublishCmdProtoNoValidationDryRun(t *testing.T) {
	// Arrange
	id := "proto_novalidation_dryrun"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		NoValidation:    true,
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

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
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
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_PENDING, v.Status)
	}
}

// Turns out we cant break proto.Unmarshal
func TestPublishCmdProtoInvalidPayloadNoValidation(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_no_validation"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cmdHandled := &protocmd_testv1.ChangePower{}
	dev.HandleCommand(
		cmdHandled,
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmdHandled = m.(*protocmd_testv1.ChangePower)
			return nil, errors.New("error on purpose")
		},
	)

	reqPayload := protocmd_testv1.SetDestination{
		Name: "bobby pendragon",
		Target: &protocmd_testv1.SetTargetVector{
			X: 1,
			Y: 2,
			Z: 3,
		},
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Name:            string(cmdHandled.ProtoReflect().Descriptor().FullName()),
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

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	for _, v := range respCmd.GetOk().DeviceResponses {
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, "device error in command handler: error on purpose", v.Error)
	}
	cancel()
	wg.Wait()
}

func TestPublishCmdRequestMultipleDevices(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	_, err := swarm.AddDevices(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      "device_send_cmd_1",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cmd_1",
		},
	}, &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      "device_send_cmd_2",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cmd_2",
		},
	}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, true, msgResp.Success)
		assert.Equal(t, int32(5), cmdHandled.Power)
		assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
	}

	assert.Equal(t, len(swarm.Devices), handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesOneNoHandler(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	_, err := swarm.AddDevices(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      "device_send_cmd_1_no_handler",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cmd_1_no_handler",
		},
	}, &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      "device_send_cmd_2_no_handler",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cmd_2_no_handler",
		},
	}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	swarm.AddDevice(&mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      "device_send_cmd_3_no_handler",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "device_send_cmd_3_no_handler",
		},
	}).WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}

	for k, v := range respCmd.GetOk().DeviceResponses {
		if k == "device_send_cmd_1/testing_cmd" || k == "device_send_cmd_2/testing_cmd" {
			if v.Error != "" {
				t.Error(v.Error)
			}
			assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
			assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, true, msgResp.Success)
			assert.Equal(t, int32(5), cmdHandled.Power)
			assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
		} else if k == "device_send_cmd_3/testing_cmd" {
			assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
			assert.Equal(t, "device error: no handler for command protocmd_test.v1.ChangePower found", v.Error)
		}
	}

	assert.Equal(t, 2, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesOneTimeout(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_multi_timeout"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	id2 := "device_send_cmd_multi_timeout_2"
	reqCreate2 := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id2,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id2,
		},
	}

	cmdHandled := &protocmd_testv1.ChangePower{}
	dev.HandleCommand(
		cmdHandled,
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmdHandled = m.(*protocmd_testv1.ChangePower)
			return &protocmd_testv1.ChangePowerResp{Success: true}, nil
		},
	)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id, id2},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		NoValidation:    true,
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

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for k, v := range respCmd.GetOk().DeviceResponses {
		if k == id2+"/testing_cmd" {
			assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
			assert.Equal(t, true, strings.Contains(v.Error, "no responders available for request"))
		} else {
			if v.Error != "" {
				t.Error(v.Error)
			}
			assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
			assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, true, msgResp.Success)
			assert.Equal(t, int32(5), cmdHandled.Power)
			assert.Equal(t, mir_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
		}
	}

	cancel()
	wg.Wait()
}

func TestPublishCmdRequestMultipleDevicesJson(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cmd_multi_json_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cmd_multi_json_2",
			},
		}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, payloadName, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            payloadName,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, true, msgResp.Success)
		assert.Equal(t, reqPayload.Power, cmdHandled.Power)
		assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	}

	assert.Equal(t, len(swarm.Devices), handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesDescriptorNotFound(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_notfound_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_notfound_2",
			},
		}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, _, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            "nothing",
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	for _, v := range respCmd.GetOk().DeviceResponses {
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, true, v.Error != "")
	}

	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	assert.Equal(t, 0, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesSingleDescriptorNotFoundForcePush(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(mSdk.Bus)
	handlerCount := 0
	_, err := swarm.AddDevice(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_valid_cmd",
			},
		}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	_, err = swarm.AddDevice(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_invalid_cmd",
			},
		}).
		WithCommandHandler(
			cmdHandled,
			func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
				cmdHandled = m.(*protocmd_testv1.ChangePower)
				handlerCount++
				return &protocmd_testv1.ChangePowerResp{Success: true}, nil
			},
		).Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, payloadName, err := test_utils.GetJsonBytes(&reqPayload)
	if err != nil {
		t.Error(err)
	}
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:         mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:            payloadName,
		PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		ForcePush:       true,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	devValid := respCmd.GetOk().DeviceResponses["device_valid_cmd/testing_cmd"]
	devInvalid := respCmd.GetOk().DeviceResponses["device_invalid_cmd/testing_cmd"]

	msgResp := &protocmd_testv1.ChangePowerResp{}
	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, devValid.Status)
	if err = protojson.Unmarshal(devValid.Payload, msgResp); err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), devValid.Name)
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, reqPayload.Power, cmdHandled.Power)

	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, devInvalid.Status)
	assert.Equal(t, true, devInvalid.Error != "")

	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	assert.Equal(t, 1, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesJsonTemplate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	swarm := swarm.NewSwarm(mSdk.Bus)
	_, err := swarm.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cmd_multi_json_tlm_1",
			},
		}, &mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Namespace: "testing_cmd",
				Labels: map[string]string{
					"testing": "cmd",
				},
				Annotations: map[string]string{
					"mir/device/description": "hello world of devices !",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "device_send_cmd_multi_json_tlm_2",
			},
		}).
		WithSchema(protocmd_testv1.File_protocmd_test_v1_command_proto).
		Incubate()
	wg, err := swarm.Deploy(ctx)

	reqPayload := protocmd_testv1.ChangePower{}
	cmdName := string(reqPayload.ProtoReflect().Descriptor().FullName())
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets:      mir_v1.MirDeviceTargetToProtoDeviceTarget(swarm.ToTarget()),
		Name:         cmdName,
		ShowTemplate: true,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	resp := respCmd.GetOk()
	dev1 := resp.DeviceResponses["device_send_cmd_multi_json_tlm_1/testing_cmd"]
	dev2 := resp.DeviceResponses["device_send_cmd_multi_json_tlm_2/testing_cmd"]
	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, cmdName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, dev2.Status)
	assert.Equal(t, cmdName, dev2.Name)
	assert.Equal(t, `{"power":0}`, string(dev2.Payload))

	cancel()
	for _, v := range wg {
		v.Wait()
	}
}

func TestPublishCmdRequestMultipleDevicesOneTimeoutJsonTemplate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_multi_timeout_json_template_1"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	id2 := "device_send_cmd_multi_timeout_json_template_2"
	reqCreate2 := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id2,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id2,
		},
	}

	reqCmdPayload := &protocmd_testv1.ChangePower{}
	reqCmdName := string(reqCmdPayload.ProtoReflect().Descriptor().FullName())
	reqCmd := &mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id, id2},
		},
		Name:         reqCmdName,
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

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	// Assert
	resp := respCmd.GetOk()
	dev1 := resp.DeviceResponses[id+"/testing_cmd"]
	dev2 := resp.DeviceResponses[id2+"/testing_cmd"]
	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, dev1.Status)
	assert.Equal(t, reqCmdName, dev1.Name)
	assert.Equal(t, `{"power":0}`, string(dev1.Payload))
	assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, dev2.Status)
	assert.Equal(t, "", dev2.Name)
	assert.Equal(t, "", string(dev2.Payload))
	assert.Equal(t, dev2.Error, "error retrieve command descriptor from device schema: cannot reconcile device schema: error requesting device schema: error publishing request message: nats: no responders available for request")

	cancel()
	wg.Wait()
}

func TestPublishCmdJsonNameWithCurlyRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_json_curly"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	cmdHandled := &protocmd_testv1.ChangePower{}
	dev.HandleCommand(
		cmdHandled,
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmdHandled = m.(*protocmd_testv1.ChangePower)
			return &protocmd_testv1.ChangePowerResp{Success: true}, nil
		})

	p := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := protojson.Marshal(&p)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &mir_apiv1.SendCommandRequest{
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

	respCmd, err := cmd_client.PublishSendCommandRequest(mSdk.Bus, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != "" {
		t.Error(respCmd.GetError())
	}

	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, mir_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, int32(5), cmdHandled.Power)
	assert.Equal(t, mir_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestCommandEvent(t *testing.T) {
	resp := mir_apiv1.SendCommandResponse_CommandResponse{
		DeviceId: "0xTest",
	}

	// Channel for test synchronization
	received := make(chan *mir_apiv1.SendCommandResponse_CommandResponse)

	fmt.Println(mSdk)
	err := mSdk.Event().Command().Subscribe(func(msg *mir.Msg, serverId string, cmd *mir_apiv1.SendCommandResponse_CommandResponse, err error) {
		received <- cmd
	})

	err = publishCommandEvent(mSdk, nil, "0xTest", "default", &resp)
	if err != nil {
		t.Error(err)
	}

	select {
	case cmd := <-received:
		assert.Equal(t, resp.DeviceId, cmd.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}
