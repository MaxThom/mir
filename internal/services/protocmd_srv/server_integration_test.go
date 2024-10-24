package protocmd_srv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/services/core_srv"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	//devicev1 "github.com/maxthom/mir/internal/services/protocmd_srv/proto_test/gen/mir/device/v1"
	protocmd_testv1 "github.com/maxthom/mir/internal/services/protocmd_srv/proto_test/gen/protocmd_test/v1"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"gotest.tools/assert"
)

var db *surrealdb.DB
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var logTest = logger.With().Str("test", "core").Logger().Level(zerolog.DebugLevel)
var lpClient influxdb2.Client
var lpWriter api.WriteAPI
var lpQuery api.QueryAPI

var b *bus.BusConn

// TODO fix token, maybe no auth
// TODO fix bug if device not started

func TestMain(m *testing.M) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println("Test Setup")

	b, db, _, _, _ = test_utils.SetupAllExternalsPanic(ctx, test_utils.ConnsInfo{
		Name:   "test_protoflux",
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
	mSdk, err = mir.Connect("test_protocmd", busUrl)
	if err != nil {
		panic(err)
	}
	protocmdSrv := NewProtoCmdServer(logTest, mSdk, mng.NewSurrealDeviceStore(db))
	go func() {
		protocmdSrv.Listen(ctx)
	}()
	coreSrv := core_srv.NewCore(logTest, b, mng.NewSurrealDeviceStore(db))
	go func() {
		coreSrv.Listen(ctx)
	}()
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	fmt.Println(" -> protocmd")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cmd",
	})
	time.Sleep(1 * time.Second)
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cmd",
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

func TestPublishCmdListRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cmd"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	reqListCmd := &cmd_apiv1.SendListCommandsRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
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

	respListCmd, err := cmd_client.PublishListCommandsRequest(b, reqListCmd)
	if err != nil {
		t.Error(err)
	} else if respListCmd.GetError() != nil {
		t.Error(respListCmd.GetError())
	}
	sb := strings.Builder{}
	for k, v := range respListCmd.GetOk().DeviceCommands {
		sb.WriteString(k + "\n")
		for i, c := range v.Commands {
			sb.WriteString(fmt.Sprintf("\t%d. %s\n", i+1, c.Name))
		}
	}
	fmt.Println(sb.String())

	// Assert
	respList, err := core_client.PublishDeviceListRequest(b, &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	} else if respList.GetError() != nil {
		t.Error(respList.GetError())
	}
	devDb := respList.GetOk().Devices[0]

	assert.Equal(t, reqCreate.DeviceId, devDb.Spec.DeviceId)

	cancel()
	wg.Wait()
}

func TestPublishCmdRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
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

	reqCmd := &cmd_apiv1.SendCommandRequest{
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

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, int32(5), cmdHandled.Power)
	assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
	cancel()
	wg.Wait()
}

func TestPublishCmdBadRequest(t *testing.T) {
	// Arrange
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{},
		},
		Name:            "",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_UNSPECIFIED,
		Payload:         []byte{},
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, 400, int(respErr.Code))
	assert.Equal(t, true, strings.Contains(respErr.Details[0], mir_models.ErrorNoDeviceTargetProvided.Error()))
	assert.Equal(t, true, strings.Contains(respErr.Details[1], mir_models.ErrorCommandNameNotProvided.Error()))
	assert.Equal(t, true, strings.Contains(respErr.Details[2], mir_models.ErrorCommandEncodingNotSpecified.Error()))
	assert.Equal(t, true, strings.Contains(respErr.Details[3], mir_models.ErrorCommandPayloadNotProvided.Error()))
}

func TestPublishCmdJsonRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd_json"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
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

	reqCmd := &cmd_apiv1.SendCommandRequest{
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

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
	}

	// Assert
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, int32(5), cmdHandled.Power)
	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}
	respErr := respCmd.GetError()

	// Assert
	assert.Equal(t, respErr.Message, mng.ErrorNoDeviceFound.Error())
}

func TestPublishCmdProtoNoValidationDryRun(t *testing.T) {
	// Arrange
	id := "proto_novalidation_dryrun"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	reqPayload := protocmd_testv1.ChangePower{
		Power: 5,
	}
	payloadBytes, err := proto.Marshal(&reqPayload)
	if err != nil {
		t.Error(err)
	}

	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		NoValidation:    true,
		DryRun:          true,
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_PENDING, v.Status)
	}
}

// Turns out we cant break proto.Unmarshal
func TestPublishCmdProtoInvalidPayloadNoValidation(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_send_cmd"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
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

	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Name:            string(cmdHandled.ProtoReflect().Descriptor().FullName()),
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

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &common_apiv1.Error{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, "error occure while processing command", msgResp.Message)
	}
	cancel()
	wg.Wait()
}

func TestPublishCmdRequestMultipleDevices(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(&core_apiv1.CreateDeviceRequest{
		DeviceId:  "device_send_cmd",
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
	}, &core_apiv1.CreateDeviceRequest{
		DeviceId:  "device_send_cmd_2",
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets:         swarm.ToTarget(),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, true, msgResp.Success)
		assert.Equal(t, int32(5), cmdHandled.Power)
		assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
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
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(&core_apiv1.CreateDeviceRequest{
		DeviceId:  "device_send_cmd_1",
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
	}, &core_apiv1.CreateDeviceRequest{
		DeviceId:  "device_send_cmd_2",
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
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
	swarm.AddDevice(&core_apiv1.CreateDeviceRequest{
		DeviceId:  "device_send_cmd_3",
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets:         swarm.ToTarget(),
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
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
			assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, true, msgResp.Success)
			assert.Equal(t, int32(5), cmdHandled.Power)
			assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
		} else if k == "device_send_cmd_3/testing_cmd" {
			assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
			msgRespErr := &common_apiv1.Error{}
			if err := proto.Unmarshal(v.Payload, msgRespErr); err != nil {
				t.Error(err)
			}
			assert.Equal(t, true, strings.Contains(strings.Join(msgRespErr.Details, " "), "no handler for command"))
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
	id := "device_send_cmd"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
		protocmd_testv1.File_protocmd_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	id2 := "device_send_cmd_2"
	reqCreate2 := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id2,
		Name:      id2,
		Namespace: "testing_cmd",
		Labels: map[string]string{
			"testing": "cmd",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
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

	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id, id2},
		},
		Name:            string(reqPayload.ProtoReflect().Descriptor().FullName()),
		PayloadEncoding: common_apiv1.Encoding_ENCODING_PROTOBUF,
		Payload:         payloadBytes,
		NoValidation:    true,
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

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for k, v := range respCmd.GetOk().DeviceResponses {
		if k == "device_send_cmd_2/testing_cmd" {
			assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
			assert.Equal(t, true, strings.Contains(v.Error, "no responders available for request"))
		} else {
			if v.Error != "" {
				t.Error(v.Error)
			}
			assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
			assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
			if err = proto.Unmarshal(v.Payload, msgResp); err != nil {
				t.Error(err)
			}
			assert.Equal(t, true, msgResp.Success)
			assert.Equal(t, int32(5), cmdHandled.Power)
			assert.Equal(t, common_apiv1.Encoding_ENCODING_PROTOBUF, respCmd.GetOk().Encoding)
		}
	}

	cancel()
	wg.Wait()
}

func TestPublishCmdRequestMultipleDevicesJson(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cmdHandled := &protocmd_testv1.ChangePower{}
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_send_cmd",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
		}, &core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_send_cmd_2",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets:         swarm.ToTarget(),
		Name:            payloadName,
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		if v.Error != "" {
			t.Error(v.Error)
		}
		assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, v.Status)
		if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
			t.Error(err)
		}
		assert.Equal(t, true, msgResp.Success)
		assert.Equal(t, reqPayload.Power, cmdHandled.Power)
		assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
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
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_send_notfound_1",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
			},
		}, &core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_send_notfound_2",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets:         swarm.ToTarget(),
		Name:            "nothing",
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	// msgResp := &protocmd_testv1.ChangePowerResp{}
	for _, v := range respCmd.GetOk().DeviceResponses {
		// assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), v.Name)
		assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, v.Status)
		assert.Equal(t, true, v.Error != "")
		// if err = protojson.Unmarshal(v.Payload, msgResp); err != nil {
		// t.Error(err)
		// }
		// assert.Equal(t, true, msgResp.Success)
		// assert.Equal(t, reqPayload.Power, cmdHandled.Power)
	}

	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
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
	swarm := swarm.NewSwarm(b)
	handlerCount := 0
	_, err := swarm.AddDevice(
		&core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_valid",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
		&core_apiv1.CreateDeviceRequest{
			DeviceId:  "device_invalid",
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cmd",
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
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets:         swarm.ToTarget(),
		Name:            payloadName,
		PayloadEncoding: common_apiv1.Encoding_ENCODING_JSON,
		Payload:         payloadBytes,
		ForcePush:       true,
	}

	// Act
	time.Sleep(1 * time.Second)

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	} else if respCmd.GetError() != nil {
		t.Error(respCmd.GetError())
	}

	// Assert
	devValid := respCmd.GetOk().DeviceResponses["device_valid/testing_cmd"]
	devInvalid := respCmd.GetOk().DeviceResponses["device_invalid/testing_cmd"]

	msgResp := &protocmd_testv1.ChangePowerResp{}
	assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS, devValid.Status)
	if err = protojson.Unmarshal(devValid.Payload, msgResp); err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(msgResp.ProtoReflect().Descriptor().FullName()), devValid.Name)
	assert.Equal(t, true, msgResp.Success)
	assert.Equal(t, reqPayload.Power, cmdHandled.Power)

	assert.Equal(t, cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR, devInvalid.Status)
	assert.Equal(t, true, devInvalid.Error != "")

	assert.Equal(t, common_apiv1.Encoding_ENCODING_JSON, respCmd.GetOk().Encoding)
	assert.Equal(t, 1, handlerCount)
	cancel()
	for _, v := range wg {
		v.Wait()
	}
}
