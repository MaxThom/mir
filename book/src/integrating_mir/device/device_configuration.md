# Device Configuration

Device configuration is a messaging flow that allows each devices to have a state. This state is saved:

- on the server's database
- on the device local storage to enable offline workflow

The configuration or properties are divided into two components: desired properties and reported properties.

- Desired properties are sent to the server from a client, written to storage and sent to devices.
- Reported properties are sent from the device to the server and are stored in the server's storage.
  - Can be sent after handling a desired propertie
  - Can be sent as standalone to report a status

Let's add properties to change the telemetry data rate and another one to report on the HVAC status.

## Editing the Schema

Let's add some desired and reported properties:

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
just proto
# or
make proto
```

## Handle the DataRate Properties

As commands, each desired property takes a callback function that is called when the property is updated.
Contrary to commands, each desired property is stored on the device local storage ensuring proper functionality in case of network issues.

On device boot up, the device request all it's desired properties from the server. If it can't receives them, it will use
it's local storage. Each handler will always be called on device start up to help initialization.

Let's update the data rate property to our device:

```go
	m.HandleProperties(&schemav1.DataRateProp{}, func(msg proto.Message) {
			cmd := msg.(*schemav1.DataRateProp)
			if cmd.Sec < 1 {
				cmd.Sec = 1
			}
			dataRate = int(cmd.Sec)
			m.Logger().Info().Msgf("data rate changed to %d", dataRate)

			if err := m.SendProperties(&schemav1.DataRateStatus{Sec: cmd.Sec}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending data rate status")
			}
		})
```

We can now receive one desired property and the device sends one reported property to confirm
the current data rate. Reported properties can be the same as the desired or entirely different.

```bash
just run
# or
make run
```

### Update the property

Let's test:

```bash
# List all available config
mir dev cfg send weather
  schema.v1.DataRateProp{}

# Show config current values
mir cfg send weather/default -n schema.v1.DataRateProp -c
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

The config cli works the same to the commands cli.

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
```

Under properties, we see the current desired and reported properties. Moreover, under status.properties,
we see at what time each property were last updated in UTC.

*! You can also update desired properties editing the twin using the different device update commands `mir dev edit weather`*

## Report the HVAC Status

To complete the example, let's add a HVAC status properties in the activate command:

```go
	m.HandleCommand(
		&schemav1.ActivateHVACCmd{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*schemav1.ActivateHVACCmd)
			m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)
			// Report HVAC is online
			if err := m.SendProperties(&schemav1.HVACStatus{Online: true}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending HVAC status")
			}

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")
				// Report HVAC is offline
				if err := m.SendProperties(&schemav1.HVACStatus{Online: false}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
			}()

			return &schemav1.ActivateHVACResp{
				Success: true,
			}, nil
		})
```

Voila! We now have a status report if the HVAC is online and offline.

```bash
just run
# or
make run
```

### Send HVAC command

```bash
# Send ActivateHVAC command to the device
mir cmd send weather_hvac -n schema.v1.ActivateHVACCmd -p '{"durationSec":10}'
# Display the twin to see HVAC status online
mir dev ls weather
# Wait 10 seconds, display the twin to see HVAC status offline
mir dev ls weather
```

You should see the updated properties in digital twin `mir dev ls weather`:

```yaml
properties:
    desired:
        schema.v1.DataRateProp:
            sec: 5
    reported:
        schema.v1.DataRateStatus:
            sec: 5
        schema.v1.HVACStatus:
            online: true
```

After 10 seconds:

```yaml
properties:
    desired:
        schema.v1.DataRateProp:
            sec: 5
    reported:
        schema.v1.DataRateStatus:
            sec: 5
        schema.v1.HVACStatus:
            online: false
```

## Complete Code

```go
package main

import (
	"context"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	schemav1 "github.com/maxthom/mir.device.buff/proto/schema/v1"
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
		Schema(schemav1.File_schema_v1_schema_proto).
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
		&schemav1.ActivateHVACCmd{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*schemav1.ActivateHVACCmd)
			m.Logger().Info().Msgf("handling command: activating HVAC for %d sec", cmd.DurationSec)
			if err := m.SendProperties(&schemav1.HVACStatus{Online: true}); err != nil {
				m.Logger().Error().Err(err).Msg("error sending HVAC status")
			}

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")
				if err := m.SendProperties(&schemav1.HVACStatus{Online: false}); err != nil {
					m.Logger().Error().Err(err).Msg("error sending HVAC status")
				}
			}()

			return &schemav1.ActivateHVACResp{
				Success: true,
			}, nil
		})

	m.HandleProperties(&schemav1.DataRateProp{}, func(msg proto.Message) {
		cmd := msg.(*schemav1.DataRateProp)
		if cmd.Sec < 1 {
			cmd.Sec = 1
		}
		dataRate = int(cmd.Sec)
		m.Logger().Info().Msgf("data rate changed to %d", dataRate)

		if err := m.SendProperties(&schemav1.DataRateStatus{Sec: cmd.Sec}); err != nil {
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
