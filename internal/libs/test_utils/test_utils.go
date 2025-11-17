package test_utils

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/external/influx"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type ConnsInfo struct {
	Name    string
	BusUrl  string
	Surreal SurrealInfo
	Influx  InfluxInfo
}

type SurrealInfo struct {
	Url  string
	User string
	Pass string
	Ns   string
	Db   string
}

type InfluxInfo struct {
	Url    string
	Token  string
	Org    string
	Bucket string
}

func SetupAllExternalsPanic(ctx context.Context, conns ConnsInfo) (*nats.Conn, *surreal.AutoReconnDB, influxdb2.Client, api.WriteAPI, api.QueryAPI) {
	b := SetupNatsConPanic(conns.BusUrl)
	s := SetupSurrealDbConnsPanic(conns.Surreal.Url, conns.Surreal.User, conns.Surreal.Pass, conns.Surreal.Ns, conns.Surreal.Db)
	c, w, q := SetupInfluxConnsPanic(ctx, conns.Influx.Url, conns.Influx.Token, conns.Influx.Org, conns.Influx.Bucket)
	return b, s, c, w, q
}

func SetupInfluxConnsPanic(ctx context.Context, url, token, org, bucket string) (influxdb2.Client, api.WriteAPI, api.QueryAPI) {
	c := influxdb2.NewClient(url, token)
	w := c.WriteAPI(org, bucket)
	q := c.QueryAPI(org)

	if err := influx.CreateOrgAndBucket(ctx, c, org, bucket); err != nil {
		panic(err)
	}

	return c, w, q
}

func SetupSurrealDbConnsPanic(url, user, pass, ns, db string) *surreal.AutoReconnDB {
	d, err := surreal.Connect(context.Background(), url, ns, db, user, pass, surreal.ConnHandler{})
	if err != nil {
		panic(err)
	}
	return d
}

func SetupNatsConPanic(url string) *nats.Conn {
	b, err := bus.New(url)
	if err != nil {
		panic(err)
	}
	return b.Conn
}

func DeleteDevicesWithLabelsPanic(b *nats.Conn, lbl map[string]string) {
	if _, err := core_client.PublishDeviceDeleteRequest(b, &mir_apiv1.DeleteDeviceRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Labels: lbl,
		},
	}); err != nil {
		panic(err)
	}
}

func CreateDevices(bus *nats.Conn, devices []*mir_apiv1.NewDevice) ([]*mir_apiv1.Device, error) {
	resp, err := core_client.PublishDevicesCreateRequest(bus,
		&mir_apiv1.CreateDevicesRequest{
			Devices: devices,
		})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != "" {
		return nil, errors.New(resp.GetError())
	}
	return resp.GetOk().Devices, nil
}

func CreateDevice(bus *nats.Conn, device *mir_apiv1.NewDevice) (*mir_apiv1.Device, error) {
	resp, err := core_client.PublishDeviceCreateRequest(bus,
		&mir_apiv1.CreateDeviceRequest{
			Device: device,
		})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != "" {
		return nil, errors.New(resp.GetError())
	}
	return resp.GetOk(), nil
}

func ExecuteTestQueryForType[T any](t *testing.T, db *surreal.AutoReconnDB, query string, vars map[string]any) T {
	result, err := surreal.Query[T](db, query, vars)
	if err != nil {
		t.Error(err)
	}
	return result
}

func strRef(s string) *string {
	return &s
}

func IsIntegratedServices() bool {
	integratedSrv := os.Getenv("MIR_TEST_INTEGRATED_SRV")
	if strings.ToLower(integratedSrv) == "true" {
		return true
	}
	return false
}

func TestLogger(component string) zerolog.Logger {
	// TODO add env var that switch level from info to debug
	return logger.With().Str("test", component).Logger().Output(zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: true,
	}).Level(zerolog.DebugLevel)
}

func ProtoToMap(p proto.Message) (map[string]any, error) {
	jsonData, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ProtoToStructPb(p proto.Message) (*structpb.Struct, error) {
	m, err := ProtoToMap(p)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}

func ProtoToPropertyMap(p proto.Message) (map[string]any, error) {
	m, err := ProtoToMap(p)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		string(p.ProtoReflect().Descriptor().FullName()): m,
	}, nil
}
