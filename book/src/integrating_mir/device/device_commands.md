# Device Commands

Device commands are request-reply messages from the server to a set of devices. You must define two protobuf messages: one for the request and one for the reply.

Let's add a command to our example to change the data rate. First, let's definine them in the schema:

```proto
message ChangeDataRate {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECOMMAND;

	int64 datarate_sec = 1;
}

message ChangeDataRateResponse {
  bool success = 1;
}
```

As you can see, instead of having the option as MESSAGE_TYPE_TELEMETRY, it is now MESSAGE_TYPE_TELECOMMAND.
This will tell the server that this message is a command and should be handled as such. The response does not need any special annotation.

Let's regenerate the schema:

```bash
protoc --proto_path=schemav1/ \
       --go_out=schemav1 \
       --go_opt=paths=source_relative \
       schemav1/schema.proto
```

Each command takes a callback function that will be called when the server sends a command to the device:

```go
m.HandleCommand(
 	&schemav1.ChangeDataRate{}, // Empty struct of the command type
 	func(c proto.Message) (proto.Message, error) {
 		cmd := c.(*schemav1.ChangeDataRate) // Cast the command to the correct type

		/* Command processing...*/

   	// Return the command response. This can be any proto message.
    // You can also return an error that will be pass back to the server and requester.
 		return &schemav1.ChangeDataRateResponse{
 			Success: true,
 		}, nil
 	})
```

Let's complete our example by adding a command handler to change the data rate of send telemetry:

```go
package main

import (
	"context"
	"log"
	"math/rand/v2"
	"mir-device/schemav1"
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
		DeviceId("weather_dev").
		Target("nats://127.0.0.1:4222").
		Schema(schemav1.File_schema_proto).
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	dataRate := int64(3)
	m.HandleCommand(
		&schemav1.ChangeDataRate{},
		func(c proto.Message) (proto.Message, error) {
			cmd := c.(*schemav1.ChangeDataRate)

			dataRate = cmd.DatarateSec
			log.Default().Printf("handling change data rate command to %d seconds\n", cmd.DatarateSec)

			return &schemav1.ChangeDataRateResponse{
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
				if err := m.SendTelemetry(&schemav1.EnvTlm{
					Ts:          time.Now().UTC().UnixNano(),
					Temperature: rand.Int32N(101),
					Pressure:    rand.Int32N(101),
					Humidity:    rand.Int32N(101),
				}); err != nil {
            log.Default().Printf("error sending telemetry: %w\n", err)
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

Our device is now sending periodic telemetry and can receive one command to change it's data rate. Let's test it:

```bash
# List all available commands
# ps: you can use 'mir command send weather_dev' to also see the available commands
mir command list weather_dev
  schemav1.ChangeDataRate{}

# If you don't see the change in your schema, add the refresh flag.
# Available on most CLI commands
mir command list weather_dev -r
  schemav1.ChangeDataRate{}

# Show command JSON template for payload
mir cmd send weather_dev/default -n schemav1.ChangeDataRate -j
  {
    "datarateSec": 0
  }

# Send command to change data rate to 5 seconds
# ps: use single quotes for easy json
mir cmd send weather_dev/default -n schemav1.ChangeDataRate -p '{"datarateSec": 5}'
  1. weather_dev/default COMMAND_RESPONSE_STATUS_SUCCESS
  schemav1.ChangeDataRateResponse
  {
    "success": true
  }

# Use pipes to pass payload
mir cmd send weather_dev/default -n schemav1.ChangeDataRate -j > ChangeDataRate.json
# Edit ChangeDataRate.json
# Send it!
cat ChangeDataRate.json | mir cmd send weather_dev/default -n schemav1.ChangeDataRate

# Interactively edit for easy interaction
# Upon quit and save, it will send the command
mir cmd send weather_dev/default -n schemav1.ChangeDataRate -e
  1. weather_dev/default COMMAND_RESPONSE_STATUS_SUCCESS
  schemav1.ChangeDataRateResponse
  {
    "success": true
  }
```

Voila! You have successfully sent a command to the device to change it's data rate.
Look at your device logs to see the data rate change. The CLI offers many more options to interact with the server, devices, do testing and validation, etc.
