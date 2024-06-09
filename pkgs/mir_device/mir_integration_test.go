package mir_device

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	core_api "github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/core"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
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
	if err := deleteTableOrRecord(db, "devices"); err != nil {
		panic(err)
	}
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
	if err := deleteTableOrRecord(db, "devices"); err != nil {
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

func deleteTableOrRecord(db *surrealdb.DB, thing string) error {
	if _, err := db.Delete(thing); err != nil {
		return err
	}
	return nil
}

func deleteDevices(t *testing.T, db *surrealdb.DB, ids []string) error {
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
