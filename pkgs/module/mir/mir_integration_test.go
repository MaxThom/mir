package mir

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	core_client "github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/libs/test_utils"
	mir_device "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/services/core"
	"github.com/nats-io/nats.go"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
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

	coreSrv := core.NewCore(log, b, db)
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

func TestRequestCreateDevice(t *testing.T) {
	// Arrange
	m, err := Connect("module_test", busUrl)
	if err != nil {
		t.Error(err)
	}
	id := "create_device_test"
	reqCreate := core_client.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}

	// Act
	var respCreate core_client.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.DeviceId, id)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")

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
	reqCreate := core_client.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	newName := "update_device_test_renamed"
	reqUpd := core_client.UpdateDeviceRequest{
		Targets: &core_client.Targets{
			Ids: []string{id},
		},
		Request: &core_client.UpdateDeviceRequest_Meta_{
			Meta: &core_client.UpdateDeviceRequest_Meta{
				Name: &newName,
			},
		},
	}

	// Act
	var respCreate core_client.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respUpd core_client.UpdateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().UpdateDevice(
		reqUpd,
		&respUpd,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.Name, id)
	assert.Equal(t, respUpd.GetOk().GetDevices()[0].Meta.Name, newName)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, fmt.Sprintf("%v", respUpd.GetError()), "<nil>")

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
	reqCreate := core_client.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqDel := core_client.DeleteDeviceRequest{
		Targets: &core_client.Targets{
			Ids: []string{id},
		},
	}

	// Act
	var respCreate core_client.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respDel core_client.DeleteDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().DeleteDevice(
		reqDel,
		&respDel,
	)); err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, respCreate.GetOk().GetDevices()[0].Meta.Name, id)
	assert.Equal(t, len(respDel.GetOk().GetDevices()), 0)
	assert.Equal(t, fmt.Sprintf("%v", respCreate.GetError()), "<nil>")
	assert.Equal(t, fmt.Sprintf("%v", respDel.GetError()), "<nil>")

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
	reqCreate := core_client.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_module",
		Labels: map[string]string{
			"testing": "module",
		},
	}
	reqList := core_client.ListDeviceRequest{
		Targets: &core_client.Targets{
			Ids: []string{id},
		},
	}

	// Act
	var respCreate core_client.CreateDeviceResponse
	if err = m.SendRequest(Resquest().V1Alpha().CreateDevice(
		reqCreate,
		&respCreate,
	)); err != nil {
		t.Error(err)
	}

	var respList core_client.ListDeviceResponse
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
