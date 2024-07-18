package mir

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	core_api "github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/api/routes"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/pkgs/device/mir/gen/proto_test"
	"github.com/maxthom/mir/services/core"
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

// IDEA methods that returns a set of subscriber to Nats using a map
// where the key is the stream subject
// IDEA functions for each services to setup and teardown
// IDEA Add unit test boilerplate
// IDEA functions for cleaning db and bus

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	// Setup
	fmt.Println("Test Setup")
	var err error
	db, b, err = setupConns()

	if err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> cleaning db")

	coreSrv := core.NewCore(log, b, db)
	go func() {
		coreSrv.Listen(ctx)
	}()
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)

	// Prepare test data
	devReq := &core_api.CreateDeviceRequest{
		DeviceId: "TestLaunchHearthbeat",
		Labels: map[string]string{
			"factory": "B",
			"model":   "xx021",
			"test":    "mir_device",
		},
		Annotations: map[string]string{
			"utility":                "air_quality",
			"mir/device/description": "hello world of devices !",
		},
	}

	if _, err := createDevices(b, []*core_api.CreateDeviceRequest{devReq}); err != nil {
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
	if _, err := deleteDevices(b, &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
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
	cancel()
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
	resp, err := core.PublishDeviceListRequest(b, &core_api.ListDeviceRequest{
		Targets: &core_api.Targets{
			Ids: []string{"TestLaunchHearthbeat"},
		},
	})
	if err != nil {
		t.Error(err)
	}
	if resp.GetError() != nil {
		t.Error(resp.GetError())
	}

	// Assert
	// Check if online and has a hearthbeat
	devTwin := resp.GetOk().Devices[0]
	assert.Equal(t, devTwin.Status.Online, true)
	devTs := core.AsGoTime(devTwin.Status.LastHearthbeat)
	assert.Equal(t, time.Now().UTC().Sub(devTs).Abs().Seconds() < 60, true)

	cancel()
	wg.Wait()
}

func TestRequestTelemetrySchema(t *testing.T) {
	// Arrange
	schemaBytes, err := marshalProtoFiles(
		proto_test.File_proto_test_command_proto,
		proto_test.File_proto_test_telemetry_proto,
	)
	if err != nil {
		t.Error(err)
	}
	mir, err := Builder().
		DeviceId("TestTelemetrySchema").
		Target("nats://127.0.0.1:4222").
		LogLevel(LogLevelInfo).
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

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := mir.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)
	resp, err := routes.PublishSchemaRetreiveRequest(b, "TestTelemetrySchema")
	if err != nil {
		t.Error(err)
	}
	if resp.GetError() != nil {
		t.Error(resp.GetError())
	}

	// Assert
	assert.Equal(t, true, bytes.Equal(schemaBytes, resp.GetSchema()))

	cancel()
	wg.Wait()
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
	executeTestQueryForType[[]core.Device](t, db,
		q, map[string]string{
			"tb": "devices",
		})
	return nil
}

func createDevices(bus *bus.BusConn, devices []*core_api.CreateDeviceRequest) ([]*core_api.CreateDeviceResponse, error) {
	responses := []*core_api.CreateDeviceResponse{}
	for _, dev := range devices {
		resp, err := core.PublishDeviceCreateRequest(bus, dev)
		responses = append(responses, resp)
		if err != nil {
			return responses, err
		}
	}
	return responses, nil
}

func deleteDevices(bus *bus.BusConn, req *core_api.DeleteDeviceRequest) (*core_api.DeleteDeviceResponse, error) {
	resp, err := core.PublishDeviceDeleteRequest(bus, req)
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

func setupConns() (*surrealdb.DB, *bus.BusConn, error) {
	// Database
	db, err := surrealdb.New("ws://127.0.0.1:8000/rpc")
	if err != nil {
		return db, nil, err
	}

	if _, err = db.Signin(map[string]any{
		"user": "root",
		"pass": "root",
	}); err != nil {
		return db, nil, err
	}

	if _, err = db.Use("global", "mir"); err != nil {
		return db, nil, err
	}

	// Bus
	b, err := bus.New("nats://127.0.0.1:4222")
	if err != nil {
		return nil, nil, err
	}

	return db, b, nil
}

func strRef(s string) *string {
	return &s
}
