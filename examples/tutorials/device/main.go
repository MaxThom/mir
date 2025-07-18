package main

import (
	"context"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	tutorial_devicev1 "github.com/maxthom/mir/examples/tutorials/device/proto/gen/tutorial_device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		//ConfigFile("./config.yaml", mir.Yaml).
		Schema(tutorial_devicev1.File_tutorial_device_v1_schema_proto).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}
	dataRate := 3

	m.HandleCommand(
		&tutorial_devicev1.ActivateHVACCmd{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*tutorial_devicev1.ActivateHVACCmd)
			m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)
			if err := m.SendProperties(&tutorial_devicev1.HVACStatus{Online: true}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending HVAC status")
			}

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")
				if err := m.SendProperties(&tutorial_devicev1.HVACStatus{Online: false}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
			}()

			return &tutorial_devicev1.ActivateHVACResp{
				Success: true,
			}, nil
		})

	m.HandleProperties(&tutorial_devicev1.DataRateProp{}, func(msg proto.Message) {
		cmd := msg.(*tutorial_devicev1.DataRateProp)
		if cmd.Sec < 1 {
			cmd.Sec = 1
		}
		dataRate = int(cmd.Sec)
		m.Logger().Info().Msgf("data rate changed to %d", dataRate)

		if err := m.SendProperties(&tutorial_devicev1.DataRateStatus{Sec: cmd.Sec}); err != nil {
			m.Logger().Error().Err(err).Msg("error sending data rate status")
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				// If context get cancelled, stop sending telemetry and
				// decrease the wait group for graceful shutdown
				wg.Done()
				return
			case <-time.After(time.Duration(dataRate) * time.Second):
				if err := m.SendTelemetry(&tutorial_devicev1.Env{
					Ts:          time.Now().UTC().UnixNano(),
					Temperature: rand.Int32N(101),
					Pressure:    rand.Int32N(101),
					Humidity:    rand.Int32N(101),
				}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending telemetry")
				}
			}
		}
	}()

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal

	cancel()
	wg.Wait()
}
