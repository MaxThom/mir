package mir

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/services/core_srv"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mir_device "github.com/maxthom/mir/pkgs/device/mir"
	mir_module_testv1 "github.com/maxthom/mir/pkgs/module/mir/proto_test/gen/mir_module_test/v1"
	"github.com/nats-io/nats.go"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

var log = logger.With().Str("test", "core").Logger()
var db *surrealdb.DB
var b *bus.BusConn
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	// Setup cons and services
	fmt.Println("Test Setup")
	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir")
	b = test_utils.SetupNatsConPanic(busUrl)
	fmt.Println(" -> bus")
	fmt.Println(" -> db")

	coreSrv := core_srv.NewCore(log, b, mng.NewSurrealDeviceStore(db))
	go func() {
		coreSrv.Listen(ctx)
	}()
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)

	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "module",
	})
	fmt.Println(" -> ready")

	// Tests
	fmt.Println("Test Executing")
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "module",
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

func TestConnectAndClose(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	// Act
	s := m.Bus.Status()
	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
	// Assert
	assert.Equal(t, s, nats.CONNECTED)
	assert.Equal(t, m.Bus.Status(), nats.CLOSED)
}

func TestSubscribeToHearthbeat(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	md, err := mir_device.Builder().
		DeviceId("TestLaunchHearthbeat2").
		Target(busUrl).
		LogLevel(mir_device.LogLevelInfo).
		Build()
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := md.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	// Act
	hearthbeatCount := 0
	if err = m.Subscribe(Stream().V1Alpha().Hearthbeat(
		func(msg *nats.Msg, s string) {
			hearthbeatCount++
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}
	time.Sleep(15 * time.Second)
	// Assert
	assert.Equal(t, hearthbeatCount > 0, true)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
	cancel()
	wg.Wait()
}

func TestEventDeviceOnline(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	md, err := mir_device.Builder().
		DeviceId("TestOnlineEvent").
		Target(busUrl).
		LogLevel(mir_device.LogLevelInfo).
		Build()
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceOnline(
		func(msg *nats.Msg, deviceId string) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	wg, err := md.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(15 * time.Second)
	// Assert
	assert.Equal(t, count > 0, true)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
	cancel()
	wg.Wait()
}

func TestEventDeviceOffline(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	md, err := mir_device.Builder().
		DeviceId("TestOfflineEvent").
		Target(busUrl).
		LogLevel(mir_device.LogLevelInfo).
		Build()
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	countOnline := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceOnline(
		func(msg *nats.Msg, deviceId string) {
			countOnline += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}
	countOffline := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceOffline(
		func(msg *nats.Msg, deviceId string) {
			countOffline += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	wg, err := md.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(15 * time.Second)
	cancel()
	wg.Wait()
	time.Sleep(45 * time.Second)

	// Assert
	assert.Equal(t, countOnline > 0, true)
	assert.Equal(t, countOffline > 0, true)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
}

func TestRequestCreateDevice(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "create_device_test"
	reqCreate := core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceCreated(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_apiv1.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}
	if respCreate.GetError() != nil {
		t.Error(respCreate.GetError())
	}

	// Assert
	time.Sleep(1 * time.Second)
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Spec.DeviceId, id)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, 1, count)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
}

func TestRequestUpdateDevice(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "update_device_test"
	reqCreate := core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	newName := "update_device_test_renamed"
	reqUpd := core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name: &newName,
		},
	}

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceUpdated(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_apiv1.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respUpd core_apiv1.UpdateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().UpdateDevice(
		reqUpd,
		&respUpd,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	time.Sleep(1 * time.Second)
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.Name, id)
	assert.Equal(t, respUpd.GetOk().GetDevices()[0].Meta.Name, newName)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, fmt.Sprintf("%v", respUpd.GetError()), "<nil>")
	assert.Equal(t, 1, count)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
}

func TestRequestDeleteDevice(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "delete_device_test"
	reqCreate := core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqDel := core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
	}

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceDeleted(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_apiv1.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respDel core_apiv1.DeleteDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().DeleteDevice(
		reqDel,
		&respDel,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	time.Sleep(1 * time.Second)
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.Name, id)
	assert.Equal(t, len(respDel.GetOk().GetDevices()), 1)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, fmt.Sprintf("%v", respDel.GetError()), "<nil>")
	assert.Equal(t, 1, count)

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
}

func TestRequestListDevice(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "list_device_test"
	reqCreate := core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqList := core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
	}

	// Act
	var respCreate core_apiv1.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respList core_apiv1.ListDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().ListDevice(
		reqList,
		&respList,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.Name, id)
	assert.Equal(t, len(respList.GetOk().GetDevices()), 1)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, fmt.Sprintf("%v", respList.GetError()), "<nil>")

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
}

func TestRequestRetrieveSchema(t *testing.T) {
	// Arrange
	id := "device_retrieve_schema"
	schemaBytes, err := marshalProtoFiles(
		mir_module_testv1.File_mir_module_test_v1_command_proto,
		mir_module_testv1.File_mir_module_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	if err != nil {
		t.Error(err)
	}
	dev, err := mir_device.Builder().
		DeviceId(id).
		Target(busUrl).
		LogLevel(mir_device.LogLevelInfo).
		TelemetrySchema(
			mir_module_testv1.File_mir_module_test_v1_command_proto,
		).
		TelemetrySchemaProto(
			protodesc.ToFileDescriptorProto(mir_module_testv1.File_mir_module_test_v1_telemetry_proto),
		).
		Build()
	if err != nil {
		t.Error(err)
	}

	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)
	var respSchema device_apiv1.SchemaRetrieveResponse
	if err = m.SendRequest(Command().V1Alpha().RequestSchema(
		id,
		&respSchema,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, true, bytes.Equal(schemaBytes, respSchema.GetSchema()))

	if err = m.Disconnect(); err != nil {
		t.Error(err)
	}
	cancel()
	wg.Wait()
}

func marshalProtoFiles(files ...protoreflect.FileDescriptor) ([]byte, error) {
	set := []*descriptorpb.FileDescriptorProto{}
	for _, f := range files {
		set = append(set, protodesc.ToFileDescriptorProto(f))
	}

	fileDescriptorSet := &descriptorpb.FileDescriptorSet{
		File: set,
	}

	bytes, err := proto.Marshal(fileDescriptorSet)
	if err != nil {
		return []byte{}, err
	}
	return bytes, nil
}
