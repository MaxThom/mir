package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"

	mir "github.com/maxthom/mir/pkgs/module/mirv2"
	"github.com/nats-io/nats.go"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Connect("example_hearthbeat", "nats://127.0.0.1:4222")
	if err != nil {
		panic(err)
	}

	if err = m.Device().Hearthbeat().Subscribe(
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Hearthbeat ", deviceId)
			msg.Ack()
		}); err != nil {
		fmt.Println(err)
	}

	sch, err := m.Device().Schema().Request("kevin")
	if err != nil {
		fmt.Println(err)
	}
	if sch != nil {
		fmt.Println(sch.GetPackageList())
	}

	cmdResp, err := m.Device().Command().RequestRaw("kevin",
		mir.ProtoCmdDesc{
			Name:    "swarm.v1.ChangePowerRequest",
			Payload: []byte{},
		})
	fmt.Println(cmdResp)

	if err = m.Device().Telemetry().Subscribe(
		func(msg *nats.Msg, deviceId string, protoMsgName string, data []byte) {
			fmt.Println("Telemetry ", deviceId, protoMsgName)
			msg.Ack()
		}); err != nil {
		fmt.Println(err)
	}

	m.Device().Subscribe(m.Device().NewSubject("report", "v1alpha", "stats"),
		func(msg *nats.Msg, deviceId string) {
			fmt.Println("Report ", deviceId, string(msg.Data))
		},
	)

	mir_signals.WaitForOsSignals(func() {
		cancel()
		m.Disconnect()
	})
}
