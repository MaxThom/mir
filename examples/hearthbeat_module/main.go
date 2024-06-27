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

// TODO module sdk integration test, add post check request with list
// TODO bug tui list search

func main() {
	_, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Connect("example_hearthbeat", "nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Stream().V1Alpha().Hearthbeat(
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Hearthbeat ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceOnline(
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Device online ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceOffline(
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Device offline ", deviceId)
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
			Meta: &core.UpdateDeviceRequest_Meta{
				Name: &newName,
			}},
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
