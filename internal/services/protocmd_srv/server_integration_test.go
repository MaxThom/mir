package protocmd_srv

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/services/core_srv"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"gotest.tools/assert"
)

var db *surrealdb.DB
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var logTest = logger.With().Str("test", "core").Logger()
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
			Db:   "mir",
		},
		Iinflux: test_utils.InfluxInfo{
			Url:    "http://localhost:8086/",
			Token:  "-NKzSScFgqhcAl-1S40otGUwuBEp8SmHoxFIYJVARrrp-a-H81Z28BfuRlUzAKVeH9-yIYXyMS0eL6TNeJfdOw==",
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

func TestPublishCmdRequest(t *testing.T) {
	// Arrange
	_, _ = context.WithCancel(context.Background())
	reqCmd := &cmd_apiv1.SendCommandRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{"cmd_send"},
		},
	}

	respCmd, err := cmd_client.PublishSendCommandRequest(b, reqCmd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	fmt.Println(respCmd)
	assert.Equal(t, true, true)

}
