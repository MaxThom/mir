package mir

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/services/core_srv"
	mir_device "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/device/mir/gen/proto_test"
	"github.com/maxthom/mir/pkgs/module/mir"
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

	coreSrv := core_srv.NewCore(log, b, db)
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
	m, err := mir.Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	// Act
	s := m.Bus.Status()
	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
	// Assert
	assert.Equal(t, s, nats.mir.ConnectED)
	assert.Equal(t, m.Bus.Status(), nats.CLOSED)
}

func TestSubscribeToHearthbeat(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
	cancel()
	wg.Wait()
}

func TestEventDeviceOnline(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
	cancel()
	wg.Wait()
}

func TestEventDeviceOffline(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
}

func TestRequestCreateDevice(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "create_device_test"
	reqCreate := core_ito.CreateDeviceRequest{
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
		func(msg *nats.Msg, deviceId string, d *core_ito.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_ito.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	time.Sleep(1 * time.Second)
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Spec.DeviceId, id)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, 1, count)

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
}

func TestRequestUpdateDevice(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "update_device_test"
	reqCreate := core_ito.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	newName := "update_device_test_renamed"
	reqUpd := core_ito.UpdateDeviceRequest{
		Targets: &core_ito.Targets{
			Ids: []string{id},
		},
		Meta: &core_ito.UpdateDeviceRequest_Meta{
			Name: &newName,
		},
	}

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceUpdated(
		func(msg *nats.Msg, deviceId string, d *core_ito.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_ito.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respUpd core_ito.UpdateDeviceResponse
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
}

func TestRequestDeleteDevice(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "delete_device_test"
	reqCreate := core_ito.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqDel := core_ito.DeleteDeviceRequest{
		Targets: &core_ito.Targets{
			Ids: []string{id},
		},
	}

	// Act
	count := 0
	if err = m.Subscribe(Event().V1Alpha().DeviceDeleted(
		func(msg *nats.Msg, deviceId string, d *core_ito.Device) {
			count += 1
			msg.Ack()
		})); err != nil {
		t.Error(err)
	}

	var respCreate core_ito.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respDel core_ito.DeleteDeviceResponse
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
}

func TestRequestListDevice(t *testing.T) {
	// Arrange
	m, err := mir.Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "list_device_test"
	reqCreate := core_ito.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqList := core_ito.ListDeviceRequest{
		Targets: &core_ito.Targets{
			Ids: []string{id},
		},
	}

	// Act
	var respCreate core_ito.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respList core_ito.ListDeviceResponse
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

	if err = m.Dismir.Connect(); err != nil {
		t.Error(err)
	}
}

func TestRequestRetrieveSchema(t *testing.T) {
	// Arrange
	id := "device_retrieve_schema"
	schemaBytes, err := marshalProtoFiles(
		proto_test.File_proto_test_command_proto,
		proto_test.File_proto_test_telemetry_proto,
	)
	if err != nil {
		t.Error(err)
	}
	dev, err := mir_device.Builder().
		DeviceId(id).
		Target(busUrl).
		LogLevel(mir_device.LogLevelInfo).
		TelemetrySchema(
			proto_test.File_proto_test_command_proto,
		).
		TelemetrySchemaProto(
			protodesc.ToFileDescriptorProto(proto_test.File_proto_test_telemetry_proto),
		).
		Build()
	if err != nil {
		t.Error(err)
	}

	m, err := mir.Connect("module_test", busUrl)
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
	var respSchema device_ito.SchemaRetrieveResponse
	if err = m.SendRequest(Command().V1Alpha().RequestSchema(
		id,
		&respSchema,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, true, bytes.Equal(schemaBytes, respSchema.GetSchema()))

	if err = m.Dismir.Connect(); err != nil {
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
