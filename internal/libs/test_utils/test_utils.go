package test_utils

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/external/influx"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
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

func SetupAllExternalsPanic(ctx context.Context, conns ConnsInfo) (*nats.Conn, *surrealdb.DB, influxdb2.Client, api.WriteAPI, api.QueryAPI) {
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

func SetupSurrealDbConnsPanic(url, user, pass, ns, db string) *surrealdb.DB {
	d, err := surrealdb.New(url)
	if err != nil {
		panic(err)
	}

	if _, err = d.SignIn(&surrealdb.Auth{
		Username: user,
		Password: pass,
	}); err != nil {
		panic(err)
	}

	if err = d.Use(ns, db); err != nil {
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

func CreateDevices(bus *nats.Conn, devices []*mir_apiv1.CreateDeviceRequest) ([]*mir_apiv1.CreateDeviceResponse, error) {
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

func ExecuteTestQueryForType[T any](t *testing.T, db *surrealdb.DB, query string, vars map[string]any) T {
	var empty T
	result, err := surrealdb.Query[T](db, query, vars)
	if err != nil {
		t.Error(err)
	}

	res := *result
	if len(res) == 0 {
		return empty
	}

	return res[0].Result
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
