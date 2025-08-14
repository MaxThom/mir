# Device Commands

Device commands are request-reply messages from the server to a set of devices.

## Editing the Schema

You must define two protobuf messages per command: one for the request and one for the reply.

Let's add a command to our example to activate a HVAC system. First, let's define them in the schema:

```proto
message ActivateHVACCmd {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECOMMAND;

	int32 duration_sec = 1;
}

message ActivateHVACResp {
  bool success = 1;
}
```

As you can see, instead of having the option as `MESSAGE_TYPE_TELEMETRY`, it is now `MESSAGE_TYPE_TELECOMMAND`.
This will tell the server that this message is a command and should be handled as such. The response does not need any special annotation.

Let's regenerate the schema:

```bash
just proto
# or
make proto
```

## Handle the Command

Each command takes a callback function that will be called when the server sends a command to the device:

```go
m.HandleCommand(
	&schemav1.ActivateHVACCmd{},
	func(msg proto.Message) (proto.Message, error) {
		cmd := msg.(*schemav1.ActivateHVACCmd) // Cast the proto.Message to the command type

		/* Command processing...*/

  	// Return the command response. This can be any proto message.
   	// You can also return an error instead that will be pass back to the server and requester.
    return &schemav1.ActivateHVACResp{
    	Success: true,
    }, nil
})
```

Let's complete our example by adding a command handler that output some logs after the duration:

```go
package main

import (
	"context"
	"math/rand/v2"
	"mir-device/schemav1"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maxthom/mir/pkgs/device/mir"
	schemav1 "github.com/maxthom/mir.device.buff/proto/gen/schema/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		LogPretty(false).
		Schema(schemav1.File_schema_proto).
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

			go func() {
				<-time.After(time.Duration(cmd.DurationSec) * time.Second)
				m.Logger().Info().Msg("turning off HVAC")
			}()

			return &schemav1.ActivateHVACResp{
				Success: true,
			}, nil
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

Rerun the code:
```sh
just run
# or
make run
```

## Send a Command

Our device is now sending periodic telemetry and can receive one command. Let's test it:

```bash
# List all available commands
mir dev cmd send weather
  schema.v1.ActivateHVACCmd{}

# Show command JSON template for payload
mir cmd send weather/default -n schema.v1.ActivateHVACCmd -j
  {
    "durationSec": 0
  }
```

Multiple ways to send a command:

```bash
# Send command to activate the HVAC
# ps: use single quotes for easy json
mir cmd send weather/default -n schema.v1.ActivateHVACCmd -p '{"durationSec": 5}'
  1. weather/default COMMAND_RESPONSE_STATUS_SUCCESS
  schema.v1.ActivateHVACResp
  {
    "success": true
  }

# Use pipes to pass payload
mir cmd send weather/default -n schema.v1.ActivateHVACCmd -j > ActivateHVACCmd.json
# Edit ActivateHVACCmd.json
# Send it!
cat ActivateHVACCmd.json | mir cmd send weather/default -n schema.v1.ActivateHVACCmd
  1. weather/default COMMAND_RESPONSE_STATUS_SUCCESS
  schema.v1.ActivateHVACResp
  {
    "success": true
  }

# Interactively edit for easy interaction
# Upon quit and save, it will send the command
mir cmd send weather/default -n schema.v1.ActivateHVACCmd -e
  1. weather/default COMMAND_RESPONSE_STATUS_SUCCESS
  schema.v1.ActivateHVACResp
  {
    "success": true
  }
```

Voila! You have successfully sent a command to the device to change it's data rate.
Look at your device logs to see the command into effect.
