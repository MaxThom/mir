package main

import (
	"context"
	"io"
	"os"
	"syscall"
	"time"

	command_devicev1 "github.com/maxthom/mir/examples/command_device/gen/command_device/v1"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Builder().
		DeviceId("0xf86cmd").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelDebug).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		TelemetrySchema(
			command_devicev1.File_command_device_v1_command_proto,
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

	m.HandleCommand(
		&command_devicev1.ChangePower{},
		func(m protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			cmd := m.(*command_devicev1.ChangePower)

			l.Info().Msg("ChangePower command received " + cmd.String())
			time.Sleep(2 * time.Second)
			l.Info().Msg("Power changed successfully")

			return &command_devicev1.ChangePowerResponse{Success: true}, nil
		},
	)

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
