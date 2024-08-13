package test_utils

import (
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/maxthom/mir/internal/clients/core_client"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/surrealdb/surrealdb.go"
)

type ConnsInfo struct {
	Name    string
	BusUrl  string
	Surreal SurrealInfo
	Iinflux InfluxInfo
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

func SetupAllExternalsPanic(conns ConnsInfo) (*mir.Mir, *bus.BusConn, *surrealdb.DB, influxdb2.Client, api.WriteAPI, api.QueryAPI) {
	m := SetupMirSdkPanic(conns.Name, conns.BusUrl)
	b := SetupNatsConPanic(conns.BusUrl)
	s := SetupSurrealDbConnsPanic(conns.Surreal.Url, conns.Surreal.User, conns.Surreal.Pass, conns.Surreal.Ns, conns.Surreal.Db)
	c, w, q := SetupInfluxConnsPanic(conns.Iinflux.Url, conns.Iinflux.Token, conns.Iinflux.Org, conns.Iinflux.Bucket)
	return m, b, s, c, w, q
}

func SetupInfluxConnsPanic(url, token, org, bucket string) (influxdb2.Client, api.WriteAPI, api.QueryAPI) {
	c := influxdb2.NewClient(url, token)
	w := c.WriteAPI(org, bucket)
	q := c.QueryAPI(org)
	return c, w, q
}

func SetupMirSdkPanic(name, url string) *mir.Mir {
	m, err := mir.Connect(name, url)
	if err != nil {
		panic(err)
	}
	return m
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
	if _, err := core_client.PublishDeviceDeleteRequest(b, &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Labels: lbl,
		},
	}); err != nil {
		panic(err)
	}
}

func CreateDevices(bus *bus.BusConn, devices []*core_api.CreateDeviceRequest) ([]*core_api.CreateDeviceResponse, error) {
	responses := []*core_api.CreateDeviceResponse{}
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
