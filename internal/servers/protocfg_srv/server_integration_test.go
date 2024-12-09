package protocfg_srv

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	protocfg_testv1 "github.com/maxthom/mir/internal/servers/protocfg_srv/proto_test/gen/protocfg_test/v1"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/surrealdb/surrealdb.go"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("cfg")
var db *surrealdb.DB
var b *bus.BusConn
var sub *nats.Subscription
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error

	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir_testing")
	b = test_utils.SetupNatsConPanic(busUrl)
	mSdk, err = mir.Connect("test_cfg", busUrl)
	if err != nil {
		panic(err)
	}
	coreSrv, err := core_srv.NewCore(log, mSdk, mng.NewSurrealDeviceStore(db))
	if err := coreSrv.Serve(); err != nil {
		panic(err)
	}
	cfgSrv, err := NewProtoCfg(log, mSdk, mng.NewSurrealDeviceStore(db))
	if err := cfgSrv.Serve(); err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cfg",
	})
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "cfg",
	})
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	coreSrv.Shutdown()
	cfgSrv.Shutdown()
	b.Close()
	db.Close()
	fmt.Println(" -> core")
	fmt.Println(" -> nats")
	fmt.Println(" -> db")

	os.Exit(exitVal)
}

func TestPublishCmdListRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg"
	s := swarm.NewSwarm(b)
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
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

	respListCfg, err := cfg_client.PublishListConfigRequest(b, &cfg_apiv1.SendListConfigRequest{
		Targets:       s.ToTarget(),
		FilterLabels:  map[string]string{},
		RefreshSchema: false,
	})
	if err != nil {
		t.Error(err)
	} else if respListCfg.GetError() != "" {
		t.Error(respListCfg.GetError())
	}

	// Assert
	dev := respListCfg.GetOk().DeviceConfigs[id+"/testing_cmd"]
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.Configs), 4)
	assert.Equal(t, dev.Configs[0].Name, "protocfg_test.v1.Config")
	assert.Equal(t, dev.Configs[0].Labels["building"], "A")
	assert.Equal(t, dev.Configs[0].Labels["floor"], "1")
	assert.Equal(t, dev.Configs[1].Name, "protocfg_test.v1.ChangePower")
	assert.Equal(t, dev.Configs[1].Labels["building"], "A")
	assert.Equal(t, dev.Configs[1].Labels["floor"], "2")
	assert.Equal(t, dev.Configs[2].Name, "protocfg_test.v1.SetTargetVector")
	assert.Equal(t, len(dev.Configs[2].Labels), 0)
	assert.Equal(t, dev.Configs[3].Name, "protocfg_test.v1.SetDestination")
	assert.Equal(t, len(dev.Configs[3].Labels), 0)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishCmdListFiltersRequest(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_list_cfg_filters"
	s := swarm.NewSwarm(b)
	if _, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_cmd",
			Labels: map[string]string{
				"testing": "cfg",
			},
			Annotations: map[string]string{
				"mir/device/description": "hello world of devices !",
			},
		},
		Spec: &core_apiv1.Spec{
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

	respListCmd, err := cfg_client.PublishListConfigRequest(b, &cfg_apiv1.SendListConfigRequest{
		Targets: s.ToTarget(),
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
	dev := respListCmd.GetOk().DeviceConfigs[id+"/testing_cmd"]
	assert.Equal(t, dev.Error, "")
	assert.Equal(t, len(dev.Configs), 1)
	assert.Equal(t, dev.Configs[0].Name, "protocfg_test.v1.ChangePower")
	assert.Equal(t, dev.Configs[0].Labels["building"], "A")
	assert.Equal(t, dev.Configs[0].Labels["floor"], "2")

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}
