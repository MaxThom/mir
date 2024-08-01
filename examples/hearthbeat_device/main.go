package main

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Builder().
		DeviceId("0xf86ea").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		Build()
	if err != nil {
		panic(err)
	}
	l := m.Logger()

	l.Info().Msg("Mir is ready for launch")
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
