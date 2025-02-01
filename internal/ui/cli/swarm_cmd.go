package cli

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"syscall"
	"time"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	swarmv1 "github.com/maxthom/mir/internal/ui/cli/gen/swarm/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

var (
	log zerolog.Logger
)

type SwarmCmd struct {
	DeviceIds []string `name:"ids" help:"Unique id for each device"`
}

func (c *SwarmCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(c.DeviceIds) == 0 {
		err.Details = append(err.Details, "missing at least one device id")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *SwarmCmd) Run(c CLI) error {
	mir_signals.Notify(syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		return fmt.Errorf("error connecting to Mir: %w", err)
	}
	defer msgBus.Close()

	logLvl := mir.LogLevelInfo
	switch l.GetLevel() {
	case zerolog.DebugLevel:
		logLvl = mir.LogLevelDebug
	case zerolog.InfoLevel:
		logLvl = mir.LogLevelInfo
	case zerolog.WarnLevel:
		logLvl = mir.LogLevelWarning
	case zerolog.ErrorLevel:
		logLvl = mir.LogLevelError
	}

	s := swarm.NewSwarm(msgBus)
	_, err = s.AddDeviceWithIds(d.DeviceIds).
		WithSchema(swarmv1.File_swarm_v1_demo_proto).
		WithCommandHandler(&swarmv1.OpenDoorRequest{}, handleOpenDoorRequest).
		WithLogLevel(logLvl).
		Incubate()
	if err != nil {
		return fmt.Errorf("error incubating swarm: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wgs, err := s.Deploy(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("error deploying swarm: %w", err)
	}

	wg := &sync.WaitGroup{}
	for _, d := range s.Devices {
		intSec := time.Duration(5)
		d.HandleCommand(&swarmv1.SendElevatorRequest{}, handleSendElevatorRequest(d))
		d.HandleProperties(&swarmv1.ChangeDataRateProp{},
			func(m proto.Message) {
				cfg := m.(*swarmv1.ChangeDataRateProp)
				intSec = time.Duration(cfg.Sec)
				d.Logger().Info().Int("rate_sec", int(cfg.Sec)).Msg("handling change data request command")
				d.SendReportedProperties(&swarmv1.ReportedProps{
					DatarateSec: cfg.Sec,
				})
			},
		)
		wg.Add(1)
		go func() {
			for {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				case <-time.After(intSec * time.Second):
					sendTelemetry(d)
				}
			}
		}()
	}

	mir_signals.WaitForOsSignals(func() {
		cancel()
		wg.Wait()
		for _, wg := range wgs {
			wg.Wait()
		}
	})

	return nil
}

func sendTelemetry(d *mir.Mir) {
	dataEnv := swarmv1.EnvironmentTlm{
		Ts: &devicev1.Timestamp{
			Seconds: time.Now().UTC().Unix(),
			Nanos:   int32(time.Now().Nanosecond()),
		},
		Temperature: rand.Int32N(101),
		Pressure:    rand.Int32N(101),
		Humidity:    rand.Int32N(101),
		WindSpeed:   rand.Int32N(101),
	}
	d.SendTelemetry(&dataEnv)

	amp := rand.Float64() * 100
	volt := rand.Float64() * 100
	dataPwr := swarmv1.PowerConsuption{
		Ts:      time.Now().UTC().UnixNano(),
		Amp:     amp,
		Voltage: volt,
		Power:   amp * volt,
	}
	d.SendTelemetry(&dataPwr)

	d.Logger().Debug().
		Int32("humidity", dataEnv.Humidity).
		Int32("temperature", dataEnv.Temperature).
		Int32("pressure", dataEnv.Pressure).
		Int32("windspeed", dataEnv.WindSpeed).
		Float64("amp", dataPwr.Amp).
		Float64("voltage", dataPwr.Voltage).
		Float64("power", dataPwr.Power).
		Msg("sending telemetry...")
}

func handleOpenDoorRequest(m proto.Message) (proto.Message, error) {
	return &swarmv1.OpenDoorResponse{
		Success: true,
	}, nil
}

func handleSendElevatorRequest(d *mir.Mir) func(proto.Message) (proto.Message, error) {
	return func(m proto.Message) (proto.Message, error) {
		cmd := m.(*swarmv1.SendElevatorRequest)
		err := d.SendReportedProperties(&swarmv1.ReportedProps{
			ElevatorFloor: cmd.Floor,
		})
		return &swarmv1.SendElevatorResponse{
			Floor: cmd.Floor,
		}, err
	}
}
