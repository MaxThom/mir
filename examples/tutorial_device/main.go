package main

import (
	"context"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	tutorial_devicev1 "github.com/maxthom/mir/examples/tutorial_device/gen/tutorial_device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		Schema(tutorial_devicev1.File_tutorial_device_v1_schema_proto).
		LogPretty(true).
		LogLevel(mir.LogLevelDebug).
		Store(mir.StoreOptions{
			Msgs: mir.StoreMsgOptions{
				MsgStorageType: mir.StorageTypePersistent,
			},
		}).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	m.HandleCommand(
		&tutorial_devicev1.ActivateHVAC{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*tutorial_devicev1.ActivateHVAC)

			if err := m.SendProperties(&tutorial_devicev1.HVACStatus{Online: true}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending HVAC status")
			}
			m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")

				if err := m.SendProperties(&tutorial_devicev1.HVACStatus{Online: false}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
			}()

			return &tutorial_devicev1.ActivateHVACResponse{
				Success: true,
			}, nil
		},
	)

	dataRate := int32(3)
	m.HandleProperties(&tutorial_devicev1.DataRateProp{}, func(msg proto.Message) {
		cmd := msg.(*tutorial_devicev1.DataRateProp)
		if cmd.Sec < 1 {
			cmd.Sec = 1
		}
		dataRate = cmd.Sec
		m.Logger().Info().Msgf("data rate changed to %d", dataRate)

		if err := m.SendProperties(&tutorial_devicev1.DataRateStatus{Sec: dataRate}); err != nil {
			m.Logger().Error().Err(err).Msg("error sending data rate status")
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			case <-time.After(time.Duration(dataRate) * time.Second):
				t, h, p := getData()
				if err := m.SendTelemetry(&tutorial_devicev1.Env{
					Ts:          time.Now().UTC().UnixNano(),
					Temperature: int32(t),
					Humidity:    int32(h),
					Pressure:    int32(p),
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

func getData() (float64, float64, float64) {
	t := time.Now().UTC()
	seconds := float64(t.Second())
	millis := float64(t.Nanosecond()) / 1e9
	timeValue := (seconds + millis) / 60.0

	temp := (math.Sin(2*math.Pi*timeValue) + 1) * 100
	hum := (math.Cos(2*math.Pi*timeValue) + 1) * 100
	pre := (math.Cos(3*math.Pi*timeValue) + 20) * 10

	return temp, hum, pre
}
