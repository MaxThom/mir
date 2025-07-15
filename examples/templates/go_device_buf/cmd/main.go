package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxthom/mir/pkgs/device/mir"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("example_device").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		//ConfigFile("./config.yaml", mir.Yaml).
		//Schema(schemav1.File_schema_v1_schema_proto).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal

	cancel()
	wg.Wait()
}
