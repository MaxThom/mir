package main

import (
	"context"
	"flag"
	"io"
	"os"
	"syscall"

	config_devicev1 "github.com/maxthom/mir/examples/config_device/gen/config_device/v1"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
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

	l.Info().Msg("Mir is ready for launch... Launching!")
	mirWg, err := m.Launch(ctx)
	if err != nil {
		l.Error().Err(err).Msg("Abort launch error")
		os.Exit(1)
	}
	l.Info().Msg("Mir is at maxq and nominal")

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
