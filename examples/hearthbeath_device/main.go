package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/maxthom/mir/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/mir_device"
)

// Look at patterns for exposing sdk.
// Builder + Optional patterns
// Maybe there is also a more
// functional approach?

// TODO log setup
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	mir := mir_device.Builder().
		DeviceId("0xf86ea").
		Target("nats://127.0.0.1:4222").
		LogLevel("debug").
		DefaultConfigFile(mir_device.Yaml).
		Build()

	mirWg, err := mir.Launch(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
