package protoflux_srv

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/services/core_srv"
	protoflux_testv1 "github.com/maxthom/mir/internal/services/protoflux_srv/proto_test/gen/protoflux_test/v1"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_models"
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

	mSdk, b, db, _, lpWriter, lpQuery = test_utils.SetupAllExternalsPanic(ctx, test_utils.ConnsInfo{
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
	protofluxSrv := NewProtoFluxServer(logTest, mSdk, mng.NewSurrealDeviceStore(db), ts.NewInfluxTelemetryStore(lpWriter))
	go func() {
		protofluxSrv.Listen(ctx)
	}()
	coreSrv := core_srv.NewCore(logTest, b, mng.NewSurrealDeviceStore(db))
	go func() {
		coreSrv.Listen(ctx)
	}()
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> ts")
	fmt.Println(" -> core")
	fmt.Println(" -> protoflux")
	time.Sleep(1 * time.Second)
	// Clear data
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "tlm",
	})
	time.Sleep(1 * time.Second)
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	test_utils.DeleteDevicesWithLabelsPanic(b, map[string]string{
		"testing": "tlm",
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

func TestPublishDevicePushTelemetry(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_push_tlm"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "tlm",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
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
	wgTlm := &sync.WaitGroup{}
	wgTlm.Add(2)
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			data := protoflux_testv1.EnvTlm{
				Temperature: int32(i * 5),
				Pressure:    int32(i * 10),
				Humidity:    int32(i * 15),
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			amp := float64(i * 5.0)
			volt := float64(i * 10.0)
			data := protoflux_testv1.PowerTlm{
				Amp:   amp,
				Volt:  volt,
				Power: amp * volt,
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	wgTlm.Wait()

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
	originalSchemaBytes, err := mir_models.MarhsalProtoFiles(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	)
	if err != nil {
		t.Error(err)
	}
	storedSchemaBytes, err := zstd.DecompressData(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}

	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "v1alpha.integration_test.telemetry.EnvTlm" or r["_measurement"] == "v1alpha.integration_test.telemetry.PowerTlm") |> filter(fn: (r) => r["deviceId"] == "device_push_tlm")`)
	if err != nil {
		t.Error(err)
	}
	dpCount := 0
	for dpResult.Next() {
		dpCount++
	}
	if dpResult.Err() != nil {
		t.Error(err)
	}

	assert.Equal(t, reqCreate.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, len(originalSchemaBytes), len(storedSchemaBytes))
	lastFetch := mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
	tspan := time.Now().UTC().Sub(lastFetch)
	assert.Equal(t, true, tspan.Seconds() < 10)
	assert.Equal(t, 48, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDeviceSchemaAlreadyPresent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_schema_present"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "tlm",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}
	compSch, err := mir_models.CompressProtoFiles(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	)
	if err != nil {
		t.Error(err)
	}
	timeFetch := time.Date(1992, 10, 14, 14, 20, 00, 00, time.UTC)
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Status: &core_apiv1.UpdateDeviceRequest_Status{
			Schema: &core_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: compSch,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	_, err = core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	wgTlm := &sync.WaitGroup{}
	wgTlm.Add(2)
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			data := protoflux_testv1.EnvTlm{
				Temperature: int32(i * 5),
				Pressure:    int32(i * 10),
				Humidity:    int32(i * 15),
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			amp := float64(i * 5.0)
			volt := float64(i * 10.0)
			data := protoflux_testv1.PowerTlm{
				Amp:   amp,
				Volt:  volt,
				Power: amp * volt,
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	wgTlm.Wait()

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
	decompSch, err := zstd.DecompressData(compSch)
	if err != nil {
		t.Error(err)
	}
	decompStoredSchemaBytes, err := zstd.DecompressData(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "v1alpha.integration_test.telemetry.EnvTlm" or r["_measurement"] == "v1alpha.integration_test.telemetry.PowerTlm") |> filter(fn: (r) => r["deviceId"] == "device_schema_present")`)
	if err != nil {
		t.Error(err)
	}
	dpCount := 0
	for dpResult.Next() {
		dpCount++
	}
	if dpResult.Err() != nil {
		t.Error(err)
	}

	assert.Equal(t, reqCreate.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, len(decompSch), len(decompStoredSchemaBytes))
	assert.Equal(t, timeFetch, mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch))
	assert.Equal(t, 48, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDeviceSchemaInvalid(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_invalid_schema"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		DeviceId:  id,
		Name:      id,
		Namespace: "testing_core",
		Labels: map[string]string{
			"testing": "tlm",
		},
		Annotations: map[string]string{
			"mir/device/description": "hello world of devices !",
		},
	}
	badSch, err := mir_models.CompressProtoFiles(
		protoflux_testv1.File_protoflux_test_v1_invalid_proto,
	)
	goodSch, err := mir_models.CompressProtoFiles(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	)
	if err != nil {
		t.Error(err)
	}
	timeFetch := time.Date(1992, 10, 14, 14, 20, 00, 00, time.UTC)
	reqUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Status: &core_apiv1.UpdateDeviceRequest_Status{
			Schema: &core_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: badSch,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).TelemetrySchema(
		protoflux_testv1.File_protoflux_test_v1_telemetry_proto,
		protoflux_testv1.File_protoflux_test_v1_command_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(b, reqCreate)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	_, err = core_client.PublishDeviceUpdateRequest(b, reqUpd)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	wg, err := dev.Launch(ctx)
	if err != nil {
		t.Error(err)
	}

	wgTlm := &sync.WaitGroup{}
	wgTlm.Add(2)
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			data := protoflux_testv1.EnvTlm{
				Temperature: int32(i * 5),
				Pressure:    int32(i * 10),
				Humidity:    int32(i * 15),
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	go func() {
		i := 0
		for i < 5 {
			time.Sleep(1 * time.Second)
			amp := float64(i * 5.0)
			volt := float64(i * 10.0)
			data := protoflux_testv1.PowerTlm{
				Amp:   amp,
				Volt:  volt,
				Power: amp * volt,
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	wgTlm.Wait()

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
	decompGoodSch, err := zstd.DecompressData(goodSch)
	if err != nil {
		t.Error(err)
	}
	decompStoredSchemaBytes, err := zstd.DecompressData(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "v1alpha.integration_test.telemetry.EnvTlm" or r["_measurement"] == "v1alpha.integration_test.telemetry.PowerTlm") |> filter(fn: (r) => r["deviceId"] == "device_invalid_schema")`)
	if err != nil {
		t.Error(err)
	}
	dpCount := 0
	for dpResult.Next() {
		dpCount++
	}
	if dpResult.Err() != nil {
		t.Error(err)
	}

	assert.Equal(t, reqCreate.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, len(decompGoodSch), len(decompStoredSchemaBytes))
	lastFetch := mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
	tspan := time.Now().UTC().Sub(lastFetch)
	assert.Equal(t, true, tspan.Seconds() < 10)
	assert.Equal(t, 48, dpCount)

	cancel()
	wg.Wait()
}
