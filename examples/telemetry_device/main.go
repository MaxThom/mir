package main

import (
	"context"
	"io"
	"math/rand/v2"
	"os"
	"syscall"
	"time"

	"github.com/maxthom/mir/examples/telemetry_device/gen"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/reflect/protodesc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	m, err := mir.Builder().
		DeviceId("0xf86ea").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelDebug).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		TelemetrySchema(
			gen.File_telemetry_proto,
		).
		TelemetrySchemaProto(
			protodesc.ToFileDescriptorProto(gen.File_command_proto),
			protodesc.ToFileDescriptorProto(gen.File_utils_proto),
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

	go func() {
		for {
			time.Sleep(3 * time.Second)
			data := gen.EnvironmentTlm{
				Temperature: rand.Int32N(101),
				Pressure:    rand.Int32N(101),
				Humidity:    rand.Int32N(101),
				WindSpeed:   rand.Int32N(101),
			}
			m.SendTelemetry(&data)
			l.Debug().Str("module", "telemetry").Any("data", data).Msg("send tlm")
		}
	}()

	go func() {
		for {
			amp := rand.Float64() * 100
			volt := rand.Float64() * 100
			time.Sleep(5 * time.Second)
			data := gen.PowerConsuption{
				Amp:     amp,
				Voltage: volt,
				Power:   amp * volt,
			}
			m.SendTelemetry(&data)
			l.Debug().Str("module", "telemetry").Any("data", data).Msg("send tlm")
		}
	}()

	go func() {
		for {
			time.Sleep(3 * time.Second)

			data := gen.Constelletion{
				Satellites: []*gen.Satellite{
					{
						Id:             "SAT-1",
						SignalStrength: 5,
						Gps: &gen.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
					{
						Id:             "SAT-2",
						SignalStrength: 4,
						Gps: &gen.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
					{
						Id:             "SAT-3",
						SignalStrength: 3,
						Gps: &gen.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
				},
			}
			m.SendTelemetry(&data)
			l.Debug().Str("module", "telemetry").Any("data", data).Msg("send tlm")
		}
	}()

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
