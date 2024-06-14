package test_utils

import (
	core_api "github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	bus "github.com/maxthom/mir/libs/external/natsio"
	"github.com/maxthom/mir/services/core"
	"github.com/surrealdb/surrealdb.go"
)

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

func strRef(s string) *string {
	return &s
}

func DeleteDevicesWithLabelsPanic(b *bus.BusConn, lbl map[string]string) {
	if _, err := core.PublishDeviceDeleteRequest(b, &core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Labels: lbl,
		},
	}); err != nil {
		panic(err)
	}
}
