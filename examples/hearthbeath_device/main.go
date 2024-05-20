package main

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/mir_device"
)

// TODO change twin
// TODO Integration test
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	mir := mir_device.Builder().
		DeviceId("0xf86ea").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir_device.LogLevelInfo).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir_device.Yaml).
		Build()
	l := mir.Logger()

	l.Info().Msg("Mir is ready for launch")
	mirWg, err := mir.Launch(ctx)
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
