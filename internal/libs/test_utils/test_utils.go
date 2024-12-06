package test_utils

import (
	"context"
	"os"
	"strings"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/external/influx"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/rs/zerolog"
	logger "github.com/rs/zerolog/log"
	"github.com/surrealdb/surrealdb.go"
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

func SetupAllExternalsPanic(ctx context.Context, conns ConnsInfo) (*bus.BusConn, *surrealdb.DB, influxdb2.Client, api.WriteAPI, api.QueryAPI) {
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

	if _, err = d.Signin(map[string]any{
		"user": user,
		"pass": pass,
	}); err != nil {
		panic(err)
	}

	if _, err = d.Use(ns, db); err != nil {
		panic(err)
	}

	return d
}

func SetupNatsConPanic(url string) *bus.BusConn {
	b, err := bus.New(url)
	if err != nil {
		panic(err)
	}
	return b
}

func DeleteDevicesWithLabelsPanic(b *bus.BusConn, lbl map[string]string) {
	if _, err := core_client.PublishDeviceDeleteRequest(b, &core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.Targets{
			Labels: lbl,
		},
	}); err != nil {
		panic(err)
	}
}

func CreateDevices(bus *bus.BusConn, devices []*core_apiv1.CreateDeviceRequest) ([]*core_apiv1.CreateDeviceResponse, error) {
	responses := []*core_apiv1.CreateDeviceResponse{}
	for _, dev := range devices {
		resp, err := core_client.PublishDeviceCreateRequest(bus, dev)
		responses = append(responses, resp)
		if err != nil {
			return responses, err
		}
	}
	return responses, nil
}

func ExecuteTestQueryForType[T any](t *testing.T, db *surrealdb.DB, query string, vars map[string]string) T {
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
