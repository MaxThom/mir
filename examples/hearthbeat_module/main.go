package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
)

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
			fmt.Println("Event device online ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceOffline(
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Event device offline ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceDeleted(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			fmt.Println("Event device deleted ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceCreated(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			fmt.Println("Event device created ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	err = m.Subscribe(mir.Event().V1Alpha().DeviceUpdated(
		func(msg *nats.Msg, deviceId string, d *core_apiv1.Device) {
			fmt.Println("Event device updated ", deviceId)
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	SendDeviceCrud(m)

	mir_signals.WaitForOsSignals(func() {
		cancel()
		m.Disconnect()
	})
}

func SendDeviceCrud(m *mir.Mir) {
	id := "CACA2MOU"
	// Request mean youre expecting a reply
	// We only have client request in the sdk
	var respCreate core_apiv1.CreateDeviceResponse
	err := m.SendRequest(mir.Resquest().V1Alpha().CreateDevice(
		core_apiv1.CreateDeviceRequest{
			DeviceId: id,
			Name:     "VRMMOU",
		},
		&respCreate,
	))
	if err != nil {
		panic(err)
	}
	if respCreate.GetError() != nil {
		fmt.Println(respCreate.GetError())
	} else {
		fmt.Println("Created device ", id)
	}

	var respList core_apiv1.ListDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().ListDevice(
		core_apiv1.ListDeviceRequest{},
		&respList,
	))
	if err != nil {
		panic(err)
	}
	if respList.GetError() != nil {
		fmt.Println(respList.GetError())
	} else {
		fmt.Println("List device ", id)
	}

	newName := "PIPI2MOU"
	var respUpd core_apiv1.UpdateDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().UpdateDevice(
		core_apiv1.UpdateDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{id},
			},
			Meta: &core_apiv1.UpdateDeviceRequest_Meta{
				Name: &newName,
			}},
		&respUpd,
	))
	if err != nil {
		panic(err)
	}
	if respUpd.GetError() != nil {
		fmt.Println(respUpd.GetError())
	} else {
		fmt.Println("Update device ", id)
	}

	var respDel core_apiv1.DeleteDeviceResponse
	err = m.SendRequest(mir.Resquest().V1Alpha().DeleteDevice(
		core_apiv1.DeleteDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{id},
			}},
		&respDel,
	))
	if err != nil {
		panic(err)
	}
	if respDel.GetError() != nil {
		fmt.Println(respDel.GetError())
	} else {
		fmt.Println("Delete device ", id)
	}
}
