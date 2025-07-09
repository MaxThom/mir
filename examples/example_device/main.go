package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	example_devicev1 "github.com/maxthom/mir/examples/example_device/gen/example_device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
)

type ExtraConfig struct {
	Sensors []Sensor `yaml:"sensors"`
}

type Sensor struct {
	Type     string        `yaml:"type"`
	Unit     string        `yaml:"unit"`
	Interval time.Duration `yaml:"interval"`
	Pass     string        `yaml:"pass" cfg:"secret"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	b := mir.Builder().
		CustomConfigFile("./cfg.yaml", mir.Yaml).
		Schema(example_devicev1.File_example_device_v1_schema_proto).
		LogPretty(true)
	cfg := ExtraConfig{}
	m, err := b.BuildWithExtraConfig(&cfg)
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	m.SendData(m.NewSubject("test", "v1", "send"), []byte("hello"), mir.Header{})

	go func() {
		for {
			time.Sleep(time.Second * 3)
			m.SendTelemetry(&example_devicev1.Env{
				Temperature: int32(24),
				Humidity:    int32(50),
				Pressure:    int32(1013),
			})
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Cancel context to gracefully shutdown
	cancel()
	wg.Wait()
}
