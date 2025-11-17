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
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	prototlm_testv1 "github.com/maxthom/mir/internal/servers/prototlm_srv/proto_test/gen/prototlm_test/v1"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	mirDevice "github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var lpClient influxdb2.Client
var lpWriter api.WriteAPI
var lpQuery api.QueryAPI
var log = test_utils.TestLogger("tlm")

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	ctx, cancel := context.WithCancel(context.Background())
	lpClient, lpWriter, lpQuery = test_utils.SetupInfluxConnsPanic(ctx, "http://localhost:8086/", "mir-operator-token", "mir", "mir_testing")
	var err error
	mSdk, err = mir.Connect("test_prototlm", busUrl)
	if err != nil {
		panic(err)
	}
	if err := dataCleanUp(); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)

	// Tests
	fmt.Println("> Test Run")
	exitVal := m.Run()
	time.Sleep(1 * time.Second)

	// Teardown
	fmt.Println("> Test Teardown")
	cancel()
	if err := mSdk.Disconnect(); err != nil {
		fmt.Println(err)
	}
	time.Sleep(1 * time.Second)

	os.Exit(exitVal)
}

func dataCleanUp() error {
	if _, err := mSdk.Client().DeleteDevice().Request(mir_v1.DeviceTarget{
		Labels: map[string]string{
			"testing": "tlm",
		},
	}); err != nil {
		return err
	}
	return nil
}

func TestPublishDevicePushTelemetry(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_push_tlm"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
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
	respList, err := core_client.PublishDeviceListRequest(mSdk.Bus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
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

	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_testing") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_push_tlm")`)
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
	lastFetch := mir_v1.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
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
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
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
	reqUpd := &mir_apiv1.UpdateDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Status: &mir_apiv1.UpdateDeviceRequest_Status{
			Schema: &mir_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: compSchBytes,
				LastSchemaFetch:  mir_v1.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).ExcludeSchemaOnLaunch().Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	_, err = core_client.PublishDeviceUpdateRequest(mSdk.Bus, reqUpd)
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
	respList, err := core_client.PublishDeviceListRequest(mSdk.Bus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
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
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_testing") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_schema_present")`)
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
	assert.Equal(t, timeFetch, mir_v1.AsGoTime(devDb.Status.Schema.LastSchemaFetch))
	assert.Equal(t, 24, dpCount)

	cancel()
	wg.Wait()
}

func TestPublishDeviceSchemaInvalid(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	id := "device_invalid_schema"
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
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
	reqUpd := &mir_apiv1.UpdateDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Status: &mir_apiv1.UpdateDeviceRequest_Status{
			Schema: &mir_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: badSchBytes,
				LastSchemaFetch:  mir_v1.AsProtoTimestamp(timeFetch),
			},
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	_, err = core_client.PublishDeviceUpdateRequest(mSdk.Bus, reqUpd)
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
	respList, err := core_client.PublishDeviceListRequest(mSdk.Bus, &mir_apiv1.ListDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
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

	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_testing") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_invalid_schema")`)
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
	lastFetch := mir_v1.AsGoTime(devDb.Status.Schema.LastSchemaFetch)
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
	reqCreate := &mir_apiv1.NewDevice{
		Meta: &mir_apiv1.Meta{
			Name:      id,
			Namespace: "testing_core",
			Labels: map[string]string{
				"testing": "tlm",
			},
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: id,
		},
	}

	dev, err := mirDevice.Builder().DeviceId(id).Store(mirDevice.StoreOptions{InMemory: true}).Target(busUrl).Schema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).Build()
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = core_client.PublishDeviceCreateRequest(mSdk.Bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: reqCreate,
		})
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
	if _, err = core_client.PublishDeviceUpdateRequest(mSdk.Bus, &mir_apiv1.UpdateDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{id},
		},
		Meta: &mir_apiv1.UpdateDeviceRequest_Meta{
			Labels: map[string]*mir_apiv1.OptString{
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
	dpResult, err := lpQuery.Query(ctx, `from(bucket: "mir_testing") |> range(start: -7s) |> filter(fn: (r) => r["_measurement"] == "prototlm_test.v1.EnvTlm" or r["_measurement"] == "prototlm_test.v1.PowerTlm") |> filter(fn: (r) => r["__id"] == "device_push_tlm_upd") |> filter(fn: (r) => r["__label_test"] == "update")`)
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
	s := swarm.NewSwarm(mSdk.Bus)
	_, err := s.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_1",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_1",
			},
		},
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_2",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_2",
			},
		}).WithSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry_proto,
	).Incubate()
	_, err = s.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_3",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_3",
			},
		},
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_4",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_4",
			},
		}).WithSchema(
		prototlm_testv1.File_prototlm_test_v1_telemetry2_proto,
	).Incubate()
	_, err = s.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_5",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_5",
			},
		},
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_6",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
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

	resp, err := tlm_client.PublishTelemetryListRequest(mSdk.Bus, &mir_apiv1.SendListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
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
	s := swarm.NewSwarm(mSdk.Bus)
	_, err := s.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_listing_!",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
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

	resp, err := tlm_client.PublishTelemetryListRequest(mSdk.Bus, &mir_apiv1.SendListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
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
	s := swarm.NewSwarm(mSdk.Bus)
	_, err := s.AddDevices(
		&mir_apiv1.NewDevice{
			Meta: &mir_apiv1.Meta{
				Name:      "dev_tlm_list_offline",
				Namespace: "testing_core",
				Labels: map[string]string{
					"testing": "tlm",
				},
			},
			Spec: &mir_apiv1.DeviceSpec{
				DeviceId: "dev_tlm_list_offline",
			},
		},
	).WithSchema(prototlm_testv1.File_prototlm_test_v1_telemetry_proto).
		Incubate()
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := tlm_client.PublishTelemetryListRequest(mSdk.Bus, &mir_apiv1.SendListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
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
