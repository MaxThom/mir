package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
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
		&core_apiv1.CreateDeviceRequest{
			Meta: &core_apiv1.Meta{
				Name: "VRMMOU",
			},
			Spec: &core_apiv1.Spec{
				DeviceId: id,
			},
		},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created device ", respCreate.Spec.DeviceId)

	respList, err := m.Server().ListDevice().Request(
		&core_apiv1.ListDeviceRequest{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("List device ", respList)

	newName := "PIPI2MOU"
	respUpd, err := m.Server().UpdateDevice().Request(
		&core_apiv1.UpdateDeviceRequest{
			Targets: &core_apiv1.DeviceTarget{
				Ids: []string{id},
			},
			Meta: &core_apiv1.UpdateDeviceRequest_Meta{
				Name: &newName,
			}},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Update device ", respUpd)

	respDel, err := m.Server().DeleteDevice().Request(
		&core_apiv1.DeleteDeviceRequest{
			Targets: &core_apiv1.DeviceTarget{
				Ids: []string{id},
			}},
	)
	fmt.Println("Delete device ", respDel)
}
