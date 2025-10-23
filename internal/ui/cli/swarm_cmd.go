package cli

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"syscall"
	"time"

	"github.com/maxthom/mir/internal/libs/boiler/mir_signals"
	"github.com/maxthom/mir/internal/services/swarm_srvc"
	"github.com/maxthom/mir/internal/ui"
	swarmv1 "github.com/maxthom/mir/internal/ui/cli/gen/swarm/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/device/mir"
	"github.com/maxthom/mir/pkgs/mir_v1"
	mSdk "github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

var (
	log zerolog.Logger
)

type SwarmCmd struct {
	DeviceIds     []string `name:"ids" help:"Unique id for each device"`
	SwarmFile     string   `short:"f" type:"path" help:"Filepath to swarm definition. You can also pipe file content. Tips: use 'mir swarm -j > swarm.yaml' to get initial content"`
	SwarmTemplate bool     `short:"j" help:"Swarm file definition with default contents"`
}

func (c *SwarmCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(c.DeviceIds) == 0 && !isPipedStdIn() && c.SwarmFile == "" && !c.SwarmTemplate {
		err.Details = append(err.Details, "No swarm definition provided via pipe or file or list of device ids")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *SwarmCmd) Run(log zerolog.Logger, m *mSdk.Mir, cfg ui.Config) error {
	if d.SwarmTemplate {
		// TODO
		return nil
	}

	ctx, cancel := mir_signals.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT)
	mirCtx, _ := cfg.GetCurrentContext()

	var swarmCfg mir_v1.Swarm
	if isPipedStdIn() || d.SwarmFile != "" {
		swarms, err := unmarshalTypeFromStdInOrFile[mir_v1.Swarm](d.SwarmFile)
		if err != nil {
			return fmt.Errorf("error reading swarm from file: %w", err)
		}
		swarmCfg = *swarms[0]
	} else {
		logLvl := mir.LogLevelInfo
		switch log.GetLevel() {
		case zerolog.DebugLevel:
			logLvl = mir.LogLevelDebug
		case zerolog.InfoLevel:
			logLvl = mir.LogLevelInfo
		case zerolog.WarnLevel:
			logLvl = mir.LogLevelWarning
		case zerolog.ErrorLevel:
			logLvl = mir.LogLevelError
		}

		swarmCfg = mir_v1.NewSwarm()
		swarmCfg.Spec.LogLevel = string(logLvl)
		swarmCfg.Spec.TelemetryFields = []mir_v1.SwarmTelemetryField{
			{
				Name: "Temperature",
				Tags: map[string]string{
					"unit": "C",
				},
				Type: mir_v1.Float32,
				Generator: &mir_v1.SwarmTelemetryGenerator{
					Expr: "cos(t*pi*2)+4",
				},
			},
			{
				Name: "Pressure",
				Tags: map[string]string{
					"unit": "Pa",
				},
				Type: mir_v1.Float32,
				Generator: &mir_v1.SwarmTelemetryGenerator{
					Expr: "cos(t*pi*2)+4",
				},
			},
			{
				Name: "Humidity",
				Tags: map[string]string{
					"unit": "%",
				},
				Type: mir_v1.Float32,
				Generator: &mir_v1.SwarmTelemetryGenerator{
					Expr: "cos(t*pi*2)+4",
				},
			},
		}

		for _, id := range d.DeviceIds {
			d := mir_v1.SwarmDevice{
				Count: 1,
				Meta: mir_v1.SwarmDeviceMeta{
					Name:      id,
					Namespace: "default",
					Annotations: map[string]string{
						"swarm": "true",
					},
				},
				Telemetry: []mir_v1.SwarmTelemetryGroup{
					{
						Name:     "Environment",
						Interval: 3 * time.Second,
						Fields: []string{
							"Temperature",
							"Humidity",
							"Pressure",
						},
					},
				},
			}
			swarmCfg.Spec.Devices = append(swarmCfg.Spec.Devices, d)
		}
		// TODO swarmCfg with ids only and a default setup
		// wgs, err = launchSwarm(ctx, m, logLvl, currentCtx, d.DeviceIds)
		// if err != nil {
		// 	cancel()
		// 	return err
		// }
	}

	swarmSvc, err := swarm_srvc.NewSwarmService(mirCtx, swarmCfg, m.Bus)
	if err != nil {
		return err
	}

	wgs, err := swarmSvc.Deploy(ctx)
	if err != nil {
		return err
	}

	mir_signals.WaitForOsSignals(func() {
		cancel()
		for _, wg := range wgs {
			wg.Wait()
		}
	})

	if err := m.Disconnect(); err != nil {
		log.Error().Err(err).Msg("error disconnecting from Mir server")
	}
	log.Info().Msg("disconnected from Mir server")

	return nil
}

// func launchSwarm(ctx context.Context, m *mSdk.Mir, logLvl mir.LogLevel, mirCtx ui.Context, ids []string) ([]*sync.WaitGroup, error) {
// 	s := swarm.NewSwarm(bus.NewWithBus(m.Bus).Conn)
// 	_, err := s.AddDeviceWithIds(ids).
// 		WithSchema(swarmv1.File_swarm_v1_demo_proto).
// 		WithLogLevel(logLvl).
// 		WithPrettyLogger(false).
// 		WithCredentials(mirCtx.Credentials).
// 		WithCerticate(mirCtx.TlsCert, mirCtx.TlsKey).
// 		WithCA(mirCtx.RootCA).
// 		Incubate()
// 	if err != nil {
// 		return nil, fmt.Errorf("error incubating swarm: %w", err)
// 	}

// 	wgs, err := s.Deploy(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("error deploying swarm: %w", err)
// 	}

// 	wg := &sync.WaitGroup{}
// 	for _, d := range s.Devices {
// 		intSec := time.Duration(5)
// 		d.HandleCommand(&swarmv1.ActivateHVAC{}, handleActivateHVACRequest(d))
// 		d.HandleProperties(&swarmv1.DataRateProp{},
// 			func(m proto.Message) {
// 				cfg := m.(*swarmv1.DataRateProp)
// 				if cfg.Sec < 1 {
// 					cfg.Sec = 1
// 				}
// 				intSec = time.Duration(cfg.Sec)
// 				d.Logger().Info().Int("rate_sec", int(cfg.Sec)).Msg("handling data rate properties")
// 				if err := d.SendProperties(&swarmv1.DataRateStatus{
// 					Sec: cfg.Sec,
// 				}); err != nil {
// 					d.Logger().Error().Err(err).Msg("error sending data rate status property")
// 				}
// 			},
// 		)
// 		wg.Add(1)
// 		go func() {
// 			for {
// 				select {
// 				case <-ctx.Done():
// 					wg.Done()
// 					return
// 				case <-time.After(intSec * time.Second):
// 					sendTelemetry(d)
// 				}
// 			}
// 		}()
// 	}

// 	return append(wgs, wg), nil
// }

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
	if err := d.SendTelemetry(&dataPwr); err != nil {
		d.Logger().Error().Err(err).Msgf("error sending telemetry: %s", err.Error())
	}

	d.Logger().Trace().
		Int32("humidity", dataEnv.Humidity).
		Int32("temperature", dataEnv.Temperature).
		Int32("pressure", dataEnv.Pressure).
		Int32("windspeed", dataEnv.WindSpeed).
		Float64("amp", dataPwr.Amp).
		Float64("voltage", dataPwr.Voltage).
		Float64("power", dataPwr.Power).
		Msg("sending telemetry...")
}

func handleActivateHVACRequest(d *mir.Mir) func(proto.Message) (proto.Message, error) {
	return func(m proto.Message) (proto.Message, error) {
		cmd := m.(*swarmv1.ActivateHVAC)

		if err := d.SendProperties(&swarmv1.HVACStatus{Online: true}); err != nil {
			d.Logger().Error().Err(err).Msgf("error sending reported properties: %s", err.Error())
		}
		d.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)

		go func() {
			<-time.After(time.Duration(cmd.DurationSec) * time.Second)
			d.Logger().Info().Msg("turning off HVAC")

			if err := d.SendProperties(&swarmv1.HVACStatus{Online: false}); err != nil {
				d.Logger().Error().Err(err).Msgf("error sending reported properties: %s", err.Error())
			}
		}()

		return &swarmv1.ActivateHVACResponse{
			Success: true,
		}, nil
	}
}
