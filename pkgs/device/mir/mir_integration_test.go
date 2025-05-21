package mir

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mir_device_testv1 "github.com/maxthom/mir/pkgs/device/mir/proto_test/gen/mir_device_test/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("device_sdk")
var db *surrealdb.DB
var b *bus.BusConn
var mSdk *mir.Mir

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error
	db, mSdk, b, err = setupConns()
	if err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> cleaning db")

	coreSrv, err := core_srv.NewCore(log, mSdk, mng.NewSurrealMirStore(db))
	if err != nil {
		panic(err)
	}
	if err = coreSrv.Serve(); err != nil {
		panic(err)
	}
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)

	// Prepare test data
	if _, err := deleteDevices(b, &mir_apiv1.DeleteDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Labels: map[string]string{
				"test": "mir_device",
			},
		},
	}); err != nil {
		panic(err)
	}
	devReq := &mir_apiv1.CreateDeviceRequest{
		Meta: &mir_apiv1.Meta{
			Labels: map[string]string{
				"factory": "B",
				"model":   "xx021",
				"test":    "mir_device",
			},
			Annotations: map[string]string{
				"utility":                "air_quality",
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "TestLaunchHearthbeat",
		},
	}

	if _, err := createDevices(b, []*mir_apiv1.CreateDeviceRequest{devReq}); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)

	fmt.Println(" -> test data prepared")
	fmt.Println(" -> ready")

	// Tests
	fmt.Println("Test Executing")
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	if _, err := deleteDevices(b, &mir_apiv1.DeleteDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Labels: map[string]string{
				"test": "mir_device",
			},
		},
	}); err != nil {
		panic(err)
	}
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	b.Close()
	db.Close()
	fmt.Println(" -> core")
	fmt.Println(" -> nats")
	fmt.Println(" -> db")

	os.Exit(exitVal)
}

func TestLaunchHearthbeat(t *testing.T) {
	// Arrange
	mir, err := Builder().
		DeviceId("TestLaunchHearthbeat").
		Store(StoreOptions{InMemory: true}).
		Target("nats://127.0.0.1:4222").
		LogLevel(LogLevelInfo).
		Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := mir.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	// Takes some time for the hearthbeat to be sent
	time.Sleep(15 * time.Second)
	resp, err := core_client.PublishDeviceListRequest(b, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{"TestLaunchHearthbeat"},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	// Check if online and has a hearthbeat
	devTwin := resp.GetOk().Devices[0]
	assert.Equal(t, devTwin.Status.Online, true)
	devTs := mir_v1.AsGoTime(devTwin.Status.LastHearthbeat)
	assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() < 60, true)

	cancel()
	wg.Wait()
}

func TestRequestTelemetrySchema(t *testing.T) {
	// Arrange
	schemaBytes, err := marshalProtoFiles(
		mir_device_testv1.File_mir_device_test_v1_command_proto,
		mir_device_testv1.File_mir_device_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	if err != nil {
		t.Error(err)
	}
	mir, err := Builder().
		DeviceId("TestTelemetrySchema").
		Store(StoreOptions{InMemory: true}).
		Target("nats://127.0.0.1:4222").
		LogLevel(LogLevelInfo).
		Schema(
			mir_device_testv1.File_mir_device_test_v1_command_proto,
		).
		SchemaProto(
			protodesc.ToFileDescriptorProto(mir_device_testv1.File_mir_device_test_v1_telemetry_proto),
		).
		Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := mir.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)
	resp, err := device_client.PublishSchemaRetrieveRequest(b, "TestTelemetrySchema")
	if err != nil {
		t.Error(err)
	}
	if resp.GetError() != "" {
		t.Error(resp.GetError())
	}

	// Assert
	assert.Equal(t, len(schemaBytes), len(resp.GetSchema()))
	assert.Equal(t, true, bytes.Equal(schemaBytes, resp.GetSchema()))

	cancel()
	wg.Wait()
}

func TestSendSchema(t *testing.T) {
	// Arrange
	devSch, err := mir_proto.NewMirProtoSchema(
		mir_device_testv1.File_mir_device_test_v1_command_proto,
		mir_device_testv1.File_mir_device_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	if err != nil {
		t.Error(err)
	}
	id := "dev_schema_send"
	mir, err := Builder().
		DeviceId(id).
		Store(StoreOptions{InMemory: true}).
		Target("nats://127.0.0.1:4222").
		LogLevel(LogLevelInfo).
		Schema(
			mir_device_testv1.File_mir_device_test_v1_command_proto,
		).
		SchemaProto(
			protodesc.ToFileDescriptorProto(mir_device_testv1.File_mir_device_test_v1_telemetry_proto),
		).
		Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := mir.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	resp, err := core_client.PublishDeviceListRequest(b, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
	})
	if err != nil {
		t.Error(err)
	}
	if resp.GetError() != "" {
		t.Error(resp.GetError())
	}
	dev := resp.GetOk().Devices[0]
	sch, err := mir_proto.DecompressSchema(dev.Status.GetSchema().CompressedSchema)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, true, mir_proto.AreSchemaEqual(sch, devSch))

	cancel()
	wg.Wait()
}

func TestStoreUpdatePropIfNew(t *testing.T) {
	id := "test_store_update_prop_if_new"
	path := "./" + id + ".db"

	s, err := NewStore(StoreOptions{
		FolderPath: path,
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	new, err := s.UpdatePropsIfNew("test", propsValue{
		LastUpdate: time.Now(),
		Value:      []byte("hello_world!"),
	})
	notNew, err := s.UpdatePropsIfNew("test", propsValue{
		LastUpdate: time.Now().Add(-5 * time.Minute),
		Value:      []byte("hello_world!"),
	})

	if err := s.Close(); err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	val, ok := s.GetProps("test")
	_, notOk := s.GetProps("test_notfound")

	if err := s.Close(); err != nil {
		t.Error(err)
	}

	assert.Equal(t, true, new)
	assert.Equal(t, false, notNew)
	assert.Equal(t, true, bytes.Equal([]byte("hello_world!"), val.Value))
	assert.Equal(t, true, ok)
	assert.Equal(t, false, notOk)
	if err := os.RemoveAll(path); err != nil {
		t.Error(err)
	}
}

func TestStoreSaveMsg(t *testing.T) {
	// Arrange
	subject := "test_sub"
	s, err := NewStore(StoreOptions{
		InMemory: true,
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	tlm := mir_device_testv1.Telemetry{
		Temperature: int32(5),
		Humidity:    int32(5),
		Pressure:    int32(5),
	}
	tlmByte, err := proto.Marshal(&tlm)
	if err != nil {
		t.Error(err)
	}
	msg := nats.Msg{
		Subject: subject,
		Data:    tlmByte,
		Header: nats.Header{
			"_msg": []string{string(tlm.ProtoReflect().Descriptor().Name())},
		},
	}

	// Act
	if err := s.SaveMsgToPending(msg); err != nil {
		t.Error(err)
	}
	msgs, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 1, len(msgs))
	assert.Equal(t, subject, msgs[0].Subject)

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}

func TestStoreSaveMsgWithTTL(t *testing.T) {
	// Arrange
	subject := "test_sub"
	s, err := NewStore(StoreOptions{
		InMemory: true,
		Msgs: StoreMsgOptions{
			RententionLimit: JsonReadableDuration(3 * time.Second),
		},
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	tlm := mir_device_testv1.Telemetry{
		Temperature: int32(5),
		Humidity:    int32(5),
		Pressure:    int32(5),
	}
	tlmByte, err := proto.Marshal(&tlm)
	if err != nil {
		t.Error(err)
	}
	msg := nats.Msg{
		Subject: subject,
		Data:    tlmByte,
		Header: nats.Header{
			"_msg": []string{string(tlm.ProtoReflect().Descriptor().Name())},
		},
	}

	// Act
	for i := 0; i < 10; i++ {
		if err := s.SaveMsgToPending(msg); err != nil {
			t.Error(err)
		}
	}
	time.Sleep(1 * time.Second)
	msgsOne, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}
	time.Sleep(3 * time.Second)
	msgsTwo, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 10, len(msgsOne))
	assert.Equal(t, 0, len(msgsTwo))

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}

func TestStoreDeleteMsgByBatch(t *testing.T) {
	// Arrange
	subject := "test_sub"
	s, err := NewStore(StoreOptions{
		InMemory: true,
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	tlm := mir_device_testv1.Telemetry{
		Temperature: int32(5),
		Humidity:    int32(5),
		Pressure:    int32(5),
	}
	tlmByte, err := proto.Marshal(&tlm)
	if err != nil {
		t.Error(err)
	}
	msg := nats.Msg{
		Subject: subject,
		Data:    tlmByte,
		Header: nats.Header{
			"_msg": []string{string(tlm.ProtoReflect().Descriptor().Name())},
		},
	}
	for i := 0; i < 10; i++ {
		if err := s.SaveMsgToPending(msg); err != nil {
			t.Error(err)
		}
	}

	// Act
	call := 0
	msgCount := 0
	if err = s.DeleteMsgByBatch(msgPendingBucket, 4, func(msgs []nats.Msg) error {
		call += 1
		msgCount += len(msgs)
		return nil
	}); err != nil {
		t.Error(err)
	}

	msgs, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 3, call)
	assert.Equal(t, 10, msgCount)
	assert.Equal(t, 0, len(msgs))

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}

func TestStoreSwapMsgByBatchLimit(t *testing.T) {
	// Arrange
	subject := "test_sub"
	s, err := NewStore(StoreOptions{
		InMemory: true,
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	tlm := mir_device_testv1.Telemetry{
		Temperature: int32(5),
		Humidity:    int32(5),
		Pressure:    int32(5),
	}
	tlmByte, err := proto.Marshal(&tlm)
	if err != nil {
		t.Error(err)
	}
	msg := nats.Msg{
		Subject: subject,
		Data:    tlmByte,
		Header: nats.Header{
			"_msg": []string{string(tlm.ProtoReflect().Descriptor().Name())},
		},
	}
	for i := 0; i < 10; i++ {
		if err := s.SaveMsgToPending(msg); err != nil {
			t.Error(err)
		}
	}

	// Act
	call := 0
	msgCount := 0
	if err = s.SwapMsgByBatch(msgPendingBucket, msgPersistentBucket, 5, func(msgs []nats.Msg) error {
		call += 1
		msgCount += len(msgs)
		return nil
	}); err != nil {
		t.Error(err)
	}

	msgsPending, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}
	msgsPersistent, err := s.ReadMsgFromPersistent()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 2, call)
	assert.Equal(t, 10, msgCount)
	assert.Equal(t, 0, len(msgsPending))
	assert.Equal(t, 10, len(msgsPersistent))

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}

func TestStoreSwapMsgByBatch(t *testing.T) {
	// Arrange
	subject := "test_sub"
	s, err := NewStore(StoreOptions{
		InMemory: true,
	})
	if err != nil {
		t.Error(err)
	}
	if err := s.Load(); err != nil {
		t.Error(err)
	}

	tlm := mir_device_testv1.Telemetry{
		Temperature: int32(5),
		Humidity:    int32(5),
		Pressure:    int32(5),
	}
	tlmByte, err := proto.Marshal(&tlm)
	if err != nil {
		t.Error(err)
	}
	msg := nats.Msg{
		Subject: subject,
		Data:    tlmByte,
		Header: nats.Header{
			"_msg": []string{string(tlm.ProtoReflect().Descriptor().Name())},
		},
	}
	for i := 0; i < 10; i++ {
		if err := s.SaveMsgToPending(msg); err != nil {
			t.Error(err)
		}
	}

	// Act
	call := 0
	msgCount := 0
	if err = s.SwapMsgByBatch(msgPendingBucket, msgPersistentBucket, 4, func(msgs []nats.Msg) error {
		call += 1
		msgCount += len(msgs)
		return nil
	}); err != nil {
		t.Error(err)
	}

	msgsPending, err := s.ReadMsgFromPending()
	if err != nil {
		t.Error(err)
	}
	msgsPersistent, err := s.ReadMsgFromPersistent()
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, 3, call)
	assert.Equal(t, 10, msgCount)
	assert.Equal(t, 0, len(msgsPending))
	assert.Equal(t, 10, len(msgsPersistent))

	if err := s.Close(); err != nil {
		t.Error(err)
	}
}
func deleteTableOrRecord(db *surrealdb.DB, thing string) error {
	if _, err := db.Delete(thing); err != nil {
		return err
	}
	return nil
}

func deleteDevicesDb(t *testing.T, db *surrealdb.DB, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("must provice at least one id")
	}

	q := "DELETE FROM type::table($tb) WHERE meta.deviceId = \""
	q += strings.Join(ids, "\" OR device_id = \"")
	q += "\";"
	executeTestQueryForType[[]mir_v1.Device](t, db,
		q, map[string]string{
			"tb": "devices",
		})
	return nil
}

func createDevices(bus *bus.BusConn, devices []*mir_apiv1.CreateDeviceRequest) ([]*mir_apiv1.CreateDeviceResponse, error) {
	responses := []*mir_apiv1.CreateDeviceResponse{}
	for _, dev := range devices {
		resp, err := core_client.PublishDeviceCreateRequest(bus, dev)
		responses = append(responses, resp)
		if err != nil {
			return responses, err
		}
	}
	return responses, nil
}

func deleteDevices(bus *bus.BusConn, req *mir_apiv1.DeleteDeviceRequest) (*mir_apiv1.DeleteDeviceResponse, error) {
	resp, err := core_client.PublishDeviceDeleteRequest(bus, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func executeTestQueryForType[T any](t *testing.T, db *surrealdb.DB, query string, vars map[string]string) T {
	result, err := db.Query(query, vars)
	if err != nil {
		t.Error(err)
	}

	res, err := surrealdb.SmartUnmarshal[T](result, err)
	if err != nil {
		t.Error(err)
	}

	return res
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

func setupConns() (*surrealdb.DB, *mir.Mir, *bus.BusConn, error) {
	// Database
	db, err := surrealdb.New("ws://127.0.0.1:8000/rpc")
	if err != nil {
		return db, nil, nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": "root",
		"pass": "root",
	}); err != nil {
		return db, nil, nil, err
	}

	if _, err = db.Use("global", "mir_testing"); err != nil {
		return db, nil, nil, err
	}

	// Bus
	b, err := bus.New("nats://127.0.0.1:4222")
	if err != nil {
		return db, nil, nil, err
	}
	m, err := mir.Connect("test_device_sdk", "nats://127.0.0.1:4222")
	if err != nil {
		return db, nil, b, err
	}

	return db, m, b, nil
}

func strRef(s string) *string {
	return &s
}
