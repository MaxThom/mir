package prototlm_srv

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	prototlm_testv1 "github.com/maxthom/mir/internal/servers/prototlm_srv/proto_test/gen/prototlm_test/v1"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/protobuf/types/descriptorpb"
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

// TODO fix bug if device not started

func TestMain(m *testing.M) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println("Test Setup")

	b, db, lpClient, lpWriter, lpQuery = test_utils.SetupAllExternalsPanic(ctx, test_utils.ConnsInfo{
		Name:   "test_prototlm",
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
	mSdk, err = mir.Connect("test_prototlm", busUrl)
	if err != nil {
		panic(err)
	}
	protofluxSrv, err := NewProtoTlmServer(logTest, mSdk, mng.NewSurrealDeviceStore(db), ts.NewInfluxTelemetryStore("Mir", "mir_integration_test", lpClient))
	if err != nil {
		panic(err)
	}
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
	fmt.Println(" -> prototlm")
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
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
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
			data := prototlm_testv1.EnvTlm{
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
			data := prototlm_testv1.PowerTlm{
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
	}
	devDb := respList.GetOk().Devices[0]
	originalSchema, err := mir_proto.NewMirProtoSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	if err != nil {
		t.Error(err)
	}
	storedSchema, err := mir_proto.DecompressSchema(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}

	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_integration_test") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_push_tlm")`)
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

	assert.Equal(t, reqCreate.Spec.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, true, mir_proto.AreSchemaEqual(originalSchema, storedSchema))
	lastFetch := mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
	tspan := time.Now().UTC().Sub(lastFetch)
	assert.Equal(t, true, tspan.Seconds() < 10)
	assert.Equal(t, 24, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDeviceSchemaAlreadyPresent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_schema_present"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}
	compSch, err := mir_proto.NewMirProtoSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	compSchBytes, err := compSch.CompressSchema()
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
				CompressedSchema: compSchBytes,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
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
			data := prototlm_testv1.EnvTlm{
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
			data := prototlm_testv1.PowerTlm{
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
	}
	devDb := respList.GetOk().Devices[0]
	decompSch, err := mir_proto.DecompressSchema(compSchBytes)
	if err != nil {
		t.Error(err)
	}
	decompStoredSchema, err := mir_proto.DecompressSchema(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_integration_test") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_schema_present")`)
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

	assert.Equal(t, reqCreate.Spec.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, true, mir_proto.AreSchemaEqual(decompSch, decompStoredSchema))
	assert.Equal(t, timeFetch, mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch))
	assert.Equal(t, 24, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDeviceSchemaInvalid(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_invalid_schema"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}
	badSch, err := mir_proto.NewMirProtoSchema(
		prototlm_testv1.File_prototlm_test_v1_invalid_proto,
	)
	badSchBytes, err := badSch.CompressSchema()
	goodSch, err := mir_proto.NewMirProtoSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
		descriptorpb.File_google_protobuf_descriptor_proto,
		devicev1.File_mir_device_v1_mir_proto,
	)
	goodSchBytes, err := goodSch.CompressSchema()
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
				CompressedSchema: badSchBytes,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
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
			data := prototlm_testv1.EnvTlm{
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
			data := prototlm_testv1.PowerTlm{
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
	}
	devDb := respList.GetOk().Devices[0]
	decompGoodSch, err := mir_proto.DecompressSchema(goodSchBytes)
	if err != nil {
		t.Error(err)
	}

	decompStoredSchema, err := mir_proto.DecompressSchema(devDb.Status.Schema.CompressedSchema)
	if err != nil {
		t.Error(err)
	}

	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_integration_test") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_invalid_schema")`)
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

	assert.Equal(t, reqCreate.Spec.DeviceId, devDb.Spec.DeviceId)
	assert.Equal(t, true, mir_proto.AreSchemaEqual(decompGoodSch, decompStoredSchema))
	lastFetch := mir_models.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
	tspan := time.Now().UTC().Sub(lastFetch)
	assert.Equal(t, true, tspan.Seconds() < 10)
	assert.Equal(t, 24, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDevicePushTelemetryDeviceUpdate(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_push_tlm_upd"
	reqCreate := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &core_apiv1.Spec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
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
			data := prototlm_testv1.EnvTlm{
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
			data := prototlm_testv1.PowerTlm{
				Amp:   amp,
				Volt:  volt,
				Power: amp * volt,
			}
			dev.SendTelemetry(&data)
			i++
		}
		wgTlm.Done()
	}()
	time.Sleep(2 * time.Second)
	str := "update"
	if _, err = core_client.PublishDeviceUpdateRequest(b, &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{id},
		},
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*common_apiv1.OptString{
				"test": {
					Value: &str,
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}
	wgTlm.Wait()

	// Assert
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_integration_test") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_push_tlm_upd") |> filter(fn: (r) => r["__label_test"] == "update")`)
	if err != nil {
		t.Error(err)
	} else if dpResult.Err() != nil {
		t.Error(dpResult.Err())
	}
	dpCount := 0
	for dpResult.Next() {
		dpCount++
	}
	if dpResult.Err() != nil {
		t.Error(err)
	}

	assert.Equal(t, 12, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishTelemetryListPairs(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(b)
	_, err := s.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_1",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_1",
			},
		},
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_2",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_2",
			},
		}).WithSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).Incubate()
	_, err = s.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_3",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_3",
			},
		},
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_4",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_4",
			},
		}).WithSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry2_proto,
	).Incubate()
	_, err = s.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_5",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_5",
			},
		},
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_6",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_6",
			},
		}).WithSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
		prototlm_testv1.File_prototlm_test_v1_telemetry2_proto,
	).Incubate()
	if err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	resp, err := tlm_client.PublishTelemetryListRequest(b, &tlm_apiv1.SendListTelemetryRequest{
		Targets: &core_apiv1.Targets{
			Ids: s.ToTarget().Ids,
		},
		RefreshSchema: true,
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	tlmResp := resp.GetOk()

	for _, dt := range tlmResp.DevicesTelemetry {
		assert.Equal(t, 2, len(dt.DevicesNamens))
		if slices.Contains(dt.DevicesNamens, "dev_tlm_list_1") &&
			slices.Contains(dt.DevicesNamens, "dev_tlm_list_2") {
			assert.Equal(t, true, true)
		} else if slices.Contains(dt.DevicesNamens, "dev_tlm_list_3") &&
			slices.Contains(dt.DevicesNamens, "dev_tlm_list_4") {
			assert.Equal(t, true, true)
		} else if slices.Contains(dt.DevicesNamens, "dev_tlm_list_5") &&
			slices.Contains(dt.DevicesNamens, "dev_tlm_list_6") {
			assert.Equal(t, true, true)
		} else {
			assert.Assert(t, true, false)
		}
	}

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishTelemetryList(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(b)
	_, err := s.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_listing_!",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_listing_1",
			},
		},
	).WithSchema(prototlm_testv1.File_prototlm_test_v1_telemetry_proto).
		Incubate()
	if err != nil {
		t.Error(err)
	}

	// Act
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}

	resp, err := tlm_client.PublishTelemetryListRequest(b, &tlm_apiv1.SendListTelemetryRequest{
		Targets: &core_apiv1.Targets{
			Ids: s.ToTarget().Ids,
		},
		RefreshSchema: true,
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	dt := resp.GetOk().DevicesTelemetry[0]

	assert.Equal(t, 1, len(dt.DevicesNamens))
	if slices.Contains(dt.DevicesNamens, "dev_tlm_listing_1") {
		assert.Equal(t, true, true)
	}
	for _, td := range dt.TlmDescriptors {
		if td.Name == "prototlm_test.v1.EnvTlm" {
			assert.Equal(t, true, slices.Contains(td.Fields, "temperature"))
			assert.Equal(t, true, slices.Contains(td.Fields, "humidity"))
			assert.Equal(t, true, slices.Contains(td.Fields, "pressure"))
			assert.Equal(t, 1, len(td.Labels))
			if k, ok := td.Labels["severity"]; ok && k == "high" {
				assert.Equal(t, true, true)
			}
		} else if td.Name == "prototlm_test.v1.PowerTlm" {
			assert.Equal(t, true, slices.Contains(td.Fields, "amp"))
			assert.Equal(t, true, slices.Contains(td.Fields, "volt"))
			assert.Equal(t, true, slices.Contains(td.Fields, "power"))
			assert.Equal(t, 2, len(td.Labels))
			if k, ok := td.Labels["building"]; ok && k == "A" {
				assert.Equal(t, true, true)
			}
			if k, ok := td.Labels["floor"]; ok && k == "2" {
				assert.Equal(t, true, true)
			}
		}
	}

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishTelemetryListError(t *testing.T) {
	// Arrange
	s := swarm.NewSwarm(b)
	_, err := s.AddDevices(
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name:      "dev_tlm_list_offline",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &core_apiv1.Spec{
				DeviceId: "dev_tlm_list_offline",
			},
		},
	).WithSchema(prototlm_testv1.File_prototlm_test_v1_telemetry_proto).
		Incubate()
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := tlm_client.PublishTelemetryListRequest(b, &tlm_apiv1.SendListTelemetryRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{"dev_tlm_list_offline"},
		},
		RefreshSchema: true,
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	dt := resp.GetOk().DevicesTelemetry[0]
	assert.Equal(t, 1, len(dt.DevicesNamens))
	if slices.Contains(dt.DevicesNamens, "dev_tlm_list_offline") {
		assert.Equal(t, true, true)
	}
	assert.Equal(t, dt.Error, "cannot reconcile device schema: error requesting device schema: error publishing request message: nats: no responders available for request")
}
