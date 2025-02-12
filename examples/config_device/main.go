package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"syscall"

	config_devicev1 "github.com/maxthom/mir/examples/config_device/gen/config_device/v1"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	deviceId := flag.String("id", "0xf86cfg", "Device ID")
	flag.Parse()

	m, err := mir.Builder().
		DeviceId(*deviceId).
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelDebug).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		Schema(
			config_devicev1.File_config_device_v1_config_proto,
		).
		Build()
	if err != nil {
		panic(err)
	}
	l := m.Logger()

	m.HandleProperties(&config_devicev1.PhoneConfig{}, func(msg proto.Message) {
		cfg := msg.(*config_devicev1.PhoneConfig)
		fmt.Println(cfg)
	})

	l.Info().Msg("Mir is ready for launch... Launching!")
	mirWg, err := m.Launch(ctx)
	if err != nil {
		l.Error().Err(err).Msg("Abort launch error")
		os.Exit(1)
	}
	l.Info().Msg("Mir is at maxq and nominal")

	if err = m.SendReportedProperties(&config_devicev1.Args{
		PizzaCount: 23,
		Names:      []string{"pep", "hawai"},
		Ingredients: map[string]int32{
			"pep":    3,
			"tomato": 1,
			"cheese": 5,
		},
	}); err != nil {
		l.Error().Err(err).Msg("Error sending reported properties")
	}

	m.HandleProperties(&config_devicev1.Config{}, func(msg proto.Message) {
		cfg := msg.(*config_devicev1.Config)
		fmt.Println(cfg)
		if err = m.SendReportedProperties(&config_devicev1.Args{
			PizzaCount: 14,
			Names:      []string{"marguerita"},
			Ingredients: map[string]int32{
				"basilic": 1,
				"cheese":  5,
			},
		}); err != nil {
			l.Error().Err(err).Msg("Error sending reported properties")
		}
	})

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
