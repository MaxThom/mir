package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"

	command_devicev1 "github.com/maxthom/mir/examples/command_device/gen/command_device/v1"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
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

	err = m.Subscribe(mir.Event().V1Alpha().DeviceCommand(
		func(msg *nats.Msg, deviceId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse) {
			fmt.Println("---EVENT---")
			msg.Ack()
		}))
	if err != nil {
		panic(err)
	}

	respList := &cmd_apiv1.SendListCommandsResponse{}
	if e := m.SendRequest(mir.Resquest().V1Alpha().ListDeviceCommands(
		&cmd_apiv1.SendListCommandsRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{"0xf86tlm", "0xf86cmd"},
			},
		}, respList)); e != nil {
		fmt.Println(e)
	}
	fmt.Println(respList)
	fmt.Println("---------")

	cmdPayload := &command_devicev1.ChangePower{}
	respProto := map[string]mir.SendDeviceCommandResponseProto{}
	if e := m.SendRequest(mir.Resquest().V1Alpha().SendDeviceCommandProto(
		&mir.SendDeviceCommandRequestProto{
			Targets: &core_apiv1.Targets{
				Ids: []string{"0xf86tlm", "0xf86cmd"},
			},
			Command: cmdPayload,
		}, respProto, &command_devicev1.ChangePowerResponse{})); e != nil {
		fmt.Println(e)
	}
	for nameNs, dev := range respProto {
		fmt.Println(nameNs, dev.DeviceId, dev.Status)
		if dev.Error != "" {
			fmt.Println(dev.Error)
			continue
		}
		fmt.Println(dev.Response.(*command_devicev1.ChangePowerResponse))
	}

	cancel()
	m.Disconnect()
}
