package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
)

// TODO generate events
// TODO listen to events
// TODO comment functions
// TODO integration test

// TODO switch core to sdk
// TODO combine tui and cli
// TODO add install of bin of 'mir' (tui/cli) in makefile to /usr/local/bin
// TODO move device id and disabled to spec

func main() {
	_, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Connect("example_hearthbeat", "nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	// We only have device streams in the sdk
	err = m.Subscribe(mir.Stream().V1Alpha().Hearthbeat(
		func(msg *nats.Msg, s string) {
			fmt.Println("Hearthbeat")
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	id := "CACA2MOU"
	// Request mean youre expecting a reply
	// We only have client request in the sdk
	var respCreate core.CreateDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().CreateDevice(
		core.CreateDeviceRequest{
			DeviceId: id,
			Name:     "VRMMOU",
		},
		&respCreate,
	))
	if err != nil {
		panic(err)
	}
	fmt.Println(respCreate.GetOk())
	fmt.Println(respCreate.GetError())

	var respList core.ListDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().ListDevice(
		core.ListDeviceRequest{},
		&respList,
	))
	if err != nil {
		panic(err)
	}
	fmt.Println(respList.GetOk())
	fmt.Println(respList.GetError())

	newName := "PIPI2MOU"
	var respUpd core.UpdateDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().UpdateDevice(
		core.UpdateDeviceRequest{
			Targets: &core.Targets{
				Ids: []string{id},
			},
			Request: &core.UpdateDeviceRequest_Meta_{
				Meta: &core.UpdateDeviceRequest_Meta{
					Name: &newName,
				}}},
		&respUpd,
	))
	if err != nil {
		panic(err)
	}
	fmt.Println(respUpd.GetOk())
	fmt.Println(respUpd.GetError())

	var respDel core.DeleteDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().DeleteDevice(
		core.DeleteDeviceRequest{
			Targets: &core.Targets{
				Ids: []string{id},
			}},
		&respDel,
	))
	if err != nil {
		panic(err)
	}
	fmt.Println(respDel.GetOk())
	fmt.Println(respDel.GetError())

	mir_signals.WaitForOsSignals(func() {
		cancel()
		m.Disconnect()
	})
}
