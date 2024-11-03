package main

import (
	"context"
	"flag"
	"io"
	"math/rand/v2"
	"os"
	"syscall"
	"time"

	telemetry_devicev1 "github.com/maxthom/mir/examples/telemetry_device/gen/telemetry_device/v1"
	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	deviceId := flag.String("id", "0xf86tlm", "Device ID")
	flag.Parse()

	m, err := mir.Builder().
		DeviceId(*deviceId).
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelDebug).
		LogWriters([]io.Writer{os.Stdout}).
		DefaultConfigFile(mir.Yaml).
		TelemetrySchema(
			telemetry_devicev1.File_telemetry_device_v1_telemetry_proto,
		).
		TelemetrySchemaProto(
			protodesc.ToFileDescriptorProto(telemetry_devicev1.File_telemetry_device_v1_utils_proto),
		).
		Build()
	if err != nil {
		panic(err)
	}
	l := m.Logger()

	m.HandleCommand(&telemetry_devicev1.ChangePower{},
		func(protoreflect.ProtoMessage) (protoreflect.ProtoMessage, error) {
			return &telemetry_devicev1.ChangePowerResponse{Success: true}, nil
		},
	)

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
			data := telemetry_devicev1.EnvironmentTlm{
				Ts: &devicev1.Timestamp{
					Seconds: time.Now().UTC().Unix(),
					Nanos:   int32(time.Now().Nanosecond()),
				},
				Temperature: rand.Int32N(101),
				Pressure:    rand.Int32N(101),
				Humidity:    rand.Int32N(101),
				WindSpeed:   rand.Int32N(101),
			}
			m.SendTelemetry(&data)
		}
	}()

	go func() {
		for {
			amp := rand.Float64() * 100
			volt := rand.Float64() * 100
			time.Sleep(5 * time.Second)
			data := telemetry_devicev1.PowerConsuption{
				Ts:      time.Now().UTC().UnixNano(),
				Amp:     amp,
				Voltage: volt,
				Power:   amp * volt,
			}
			m.SendTelemetry(&data)
		}
	}()

	go func() {
		for {
			time.Sleep(3 * time.Second)

			data := telemetry_devicev1.Constelletion{
				Ts: time.Now().UTC().Unix(),
				Satellites: []*telemetry_devicev1.Satellite{
					{
						Id:             "SAT-1",
						SignalStrength: 5,
						Gps: &telemetry_devicev1.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
					{
						Id:             "SAT-2",
						SignalStrength: 4,
						Gps: &telemetry_devicev1.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
					{
						Id:             "SAT-3",
						SignalStrength: 3,
						Gps: &telemetry_devicev1.Gps{
							Latitude:  rand.Float64() * 100,
							Longitude: rand.Float64() * 100,
							Altitude:  rand.Float64() * 1000,
						},
					},
				},
			}
			m.SendTelemetry(&data)
			//	l.Debug().Str("module", "telemetry").Any("data", data).Msg("send tlm")
		}
	}()

	mir_signals.WaitForOsSignals(func() {
		cancel()
		mirWg.Wait()
	})
}
