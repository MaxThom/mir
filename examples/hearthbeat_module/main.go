package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Connect("example_hearthbeat", "nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	err = m.Device().Hearthbeat().Subscribe("",
		func(msg *mir.Msg, deviceId string) {
			fmt.Println("Hearthbeat ", deviceId)
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	err = m.Event().DeviceOnline().Subscribe(
		func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
			fmt.Println("Event device online ", device.Spec.DeviceId, err)
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	err = m.Event().DeviceOffline().Subscribe(
		func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
			fmt.Println("Event device offline ", device.Spec.DeviceId)
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	err = m.Event().DeviceDelete().Subscribe(
		func(msg *mir.Msg, serverId string, device mir_models.Device, err error) {
			fmt.Println("Event device deleted ", device.Spec.DeviceId)
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	err = m.Event().DeviceCreate().Subscribe(
		func(msg *mir.Msg, serverId string, d mir_models.Device, err error) {
			fmt.Println("Event device created ", d.Spec.DeviceId, err)
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	err = m.Event().DeviceUpdate().Subscribe(
		func(msg *mir.Msg, serverId string, d mir_models.Device, err error) {
			fmt.Println("Event device updated ", d.Spec.DeviceId, err)
			msg.Ack()
		})
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
	respCreate, err := m.Server().CreateDevice().Request(
		mir_models.NewDevice().WithMeta(mir_models.Meta{
			Name: "VRMMOU",
		}).WithSpec(mir_models.DeviceSpec{
			DeviceId: id,
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created device ", respCreate.Spec.DeviceId)

	respList, err := m.Server().ListDevice().Request(
		mir_models.DeviceTarget{}, false,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("List device ", respList)

	newName := "PIPI2MOU"
	respUpd, err := m.Server().UpdateDevice().Request(
		mir_models.DeviceTarget{
			Ids: []string{id},
		}, mir_models.NewDevice().WithMeta(mir_models.Meta{
			Name: newName,
		}))
	if err != nil {
		panic(err)
	}
	fmt.Println("Update device ", respUpd)

	respDel, err := m.Server().DeleteDevice().Request(
		mir_models.DeviceTarget{
			Ids: []string{id},
		},
	)
	fmt.Println("Delete device ", respDel)
}
