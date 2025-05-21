package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"

	command_devicev1 "github.com/maxthom/mir/examples/command_device/gen/command_device/v1"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/module/mir"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Connect("example_hearthbeat", "nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	err = m.Event().Command().Subscribe(
		func(msg *mir.Msg, serverId string, cmd *mir_apiv1.SendCommandResponse_CommandResponse, err error) {
			fmt.Println("---EVENT---")
			msg.Ack()
		})
	if err != nil {
		panic(err)
	}

	respList, e := m.Server().ListCommands().Request(
		&mir_apiv1.SendListCommandsRequest{
			Targets: &mir_apiv1.DeviceTarget{
				Ids: []string{"0xf86tlm", "0xf86cmd"},
			},
		})
	if e != nil {
		fmt.Println(e)
	} else {
		fmt.Println(respList)
		fmt.Println("---------")
	}

	cmdPayload := &command_devicev1.ChangePower{}
	respProto, e := m.Server().SendCommand().RequestProto(
		&mir.SendDeviceCommandRequestProto{
			Targets: &mir_apiv1.DeviceTarget{
				Ids: []string{"0xf86tlm", "0xf86cmd"},
			},
			Command: cmdPayload,
		})
	if e != nil {
		fmt.Println(e)
	}
	for nameNs, dev := range respProto {
		fmt.Println(nameNs, dev.DeviceId, dev.Status)
		if dev.Error != "" {
			fmt.Println(dev.Error)
			continue
		}
	}

	cancel()
	m.Disconnect()
}
