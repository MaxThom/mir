# Device Configuration

Device configuration is a messaging flow that allows each devices to have a state.
This state enables configuration and monitoring of devices by dividing them into two components: desired properties and reported properties.

Desired properties are sent to the server from a client, written to storage and sent to devices.
Reported properties are sent from the device to the server and are stored in the server's storage.


Let's add some desired and reported properties to our schema:

```go
message DataRateProp {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECONFIG;

  int32 sec = 1;
}

message DataRateStatus {
  int32 sec = 1;
}

message HVACStatus {
   bool online = 1;
}
```

You can see the new proto annotation: `MESSAGE_TYPE_TELECONFIG`. Regenerate the schema:

```bash
protoc --proto_path=schemav1/ \
       --go_out=schemav1 \
       --go_opt=paths=source_relative \
       schemav1/schema.proto
```

As commands, each desired property takes a callback function that is called when the property is updated.
Contrary to commands, each desired property is stored on the device local storage ensuring proper functionality in case of network issues.

On device boot up, the device request all it's desired properties from the server. If it can't receives them, it will use
it's local storage. Each handler will always be called on device start up to help initialization.

Let's add a data rate property to our device:

```go
  dataRate = int32(3)
	m.HandleProperties(&schemav1.DataRateProp{}, func(msg proto.Message) {
			cmd := msg.(*schemav1.DataRateProp)
			if cmd.Sec < 1 {
				cmd.Sec = 1
			}
			dataRate = cmd.Sec
			m.Logger().Info().Msgf("data rate changed to %d", dataRate)

			if err := m.SendProperties(&schemav1.DataRateStatus{Sec: dataRate}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending data rate status")
			}
		})
```

We can now receive one desired property and the device sends one reported property to confirm
the current data rate. Reported properties can be the same as the desired or entirely different.

Let's test:

```bash
# List all available config
# ps: you can use 'mir config send weather' to also see the available config
mir config list weather
  schemav1.DataRateProp{}


# Show config current values
mir cfg send weather/default -n schemav1.DataRateProp -c
  {
    "sec": 0
  }

# Send config to change data rate to 5 seconds
mir cfg send weather/default -n schemav1.DataRateProp -e
  schemav1.DataRateProp
  {
    "sec": 5
  }
```

The config cli works similarly to the commands cli.

Let's take a look at the device twin `mir dev ls weather/default`:

```yaml
apiVersion: mir/v1alpha
kind: device
meta:
name: weather
...
properties:
  desired:
      schema.v1.DataRateProp:
          sec: 5
  reported:
      schema.v1.DataRateStatus:
          sec: 5
      schema.v1.HVACStatus:
          online: false
status:
...
  properties:
      desired:
          schema.v1.DataRateProp: 2025-02-15T17:01:25.686135311Z
      reported:
          schema.v1.DataRateStatus: 2025-02-15T17:01:25.689587722Z
          schema.v1.HVACStatus: 2025-02-15T16:00:07.744362086Z
```

Under properties, we see the current desired and reported properties. Moreover, under status.properties,
we see at what time each property were last updated in UTC.

*! You can also update desired properties editing the twin using the different device update commands `mir dev edit weather`*

To complete the example, let's add a HVAC status in the activate command:

```go
	m.HandleCommand(
			&schemav1.ActivateHVAC{},
			func(msg proto.Message) (proto.Message, error) {
				cmd := msg.(*schemav1.ActivateHVAC)

				if err := m.SendProperties(&schemav1.HVACStatus{Online: true}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
				m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)

				go func() {
					<-time.After(time.Duration(cmd.DurationSec) * time.Second)
					m.Logger().Info().Msg("turning off HVAC")

					// Reported properties can always be sent anywhere in the code
					if err := m.SendProperties(&schemav1.HVACStatus{Online: false}); err != nil {
						m.Logger().Error().Err(err).Msg("error sending HVAC status")
					}
				}()

				return &schemav1.ActivateHVACResponse{
					Success: true,
				}, nil
			},
		)
```

Voila! We now have a status report if the HVAC is online and offline. See it in action:

```bash
# Send ActivateHVAC command to the device
mir cmd send weather_hvac -n schema.v1.ActivateHVAC -p '{"durationSec":5}'
# Display the twin to see HVAC status Online
mir dev ls weather
# Wait 5 seconds, display the twin to see HVAC status Offline
mir dev ls weather
```

Full code example:

```go
package main

import (
	"context"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maxthom/mir/pkgs/device/mir"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		LogPretty(false).
		Schema(schemav1.File_schema.v1_schema_proto).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	m.HandleCommand(
		&schemav1.ActivateHVAC{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*schemav1.ActivateHVAC)

			if err := m.SendProperties(&schemav1.HVACStatus{Online: true}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending HVAC status")
			}
			m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")

				if err := m.SendProperties(&schemav1.HVACStatus{Online: false}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
			}()

			return &schemav1.ActivateHVACResponse{
				Success: true,
			}, nil
		},
	)

	dataRate := int32(3)
	m.HandleProperties(&schemav1.DataRateProp{}, func(msg proto.Message) {
		cmd := msg.(*schemav1.DataRateProp)
		if cmd.Sec < 1 {
			cmd.Sec = 1
		}
		dataRate = cmd.Sec
		m.Logger().Info().Msgf("data rate changed to %d", dataRate)

		if err := m.SendProperties(&schemav1.DataRateStatus{Sec: dataRate}); err != nil {
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
				if err := m.SendTelemetry(&schemav1.Env{
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
```

Voila! You have set up desired and reported properties for your device.
They are powerful tools that allow you to control and monitor your device's behavior.

Use the Config CLI to manage your device's properties.
