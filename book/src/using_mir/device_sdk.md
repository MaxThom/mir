# Mir Device Tutorial

This tutorial will guide you through the process of creating a new device using the Mir Device SDK.

To work properly, you need to have the Mir Server up & running and the Mir CLI ready to be used. Follow the [Mir Server Tutorial]().

## Initialize Go project

```bash
go mod init github.com/<user/org>/<project>
```

## Access Mir Device SDK

Go packages are managed in GitHub repository.
Since the repository is private, you need to adjust your git configuration before we can execute this line.

```bash
go get github.com/maxthom/mir/
```

First of, we need to tell git to use the SSH protocol to access the GitHub repository.

```bash
# In ~/.gitconfig
[url "ssh://git@github.com/"]
  insteadOf = https://github.com
```

Even though packages are stored in Git repository, they get downloaded through Go mirror.
Therefor, we must tell Go to download it directly from the Git repository.

```bash
go env -w GOPRIVATE=github.com/maxthom/mir
```

If any import match the pattern `github.com/maxthom/mir/*`, Go will download the package directly from the Git repository.

Now, you can run

```bash
go get github.com/maxthom/mir/
```

Ready to roll!

## Mir tooling

Mir requires a set of utility tools to properly create devices:

- [buf](https://github.com/air-verse/air): A modern protobuf schema manager.
- [protoc](https://github.com/bufbuild/buf/): The protobuf compiler.

They can be installed via `go install` or using Mir CLI:

```bash
# Mir CLI
mir tools install

# Manually
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/buf/cmd/buf@latest
```

## Anatomy of a Mir Device

At the core of the devices, it is the device unique identifier or `deviceId`.

It is the responsibility of the dev/user to manage those ids. In a near future, the system will provide different generation methods and helpers such as UUID, MAC address, load from files, etc.
In this example, the deviceId will be `weather_dev`.

The SDK use a builder pattern to get started:

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxthom/mir/pkgs/device/mir"
)

func main() {
	// Build device with different options
	// For deviceId, you can hardcode, use flags, use configuration files, etc
	m, err := mir.Builder().
		DeviceId("weather_dev").
		Target("nats://127.0.0.1:4222").
		Build()
	if err != nil {
		panic(err)
	}

	// When ready, connect to the Mir server
	ctx, cancel := context.WithCancel(context.Background())
	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	// We will add here...

	// Wait for system signal to stop the device
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal

	cancel()
	wg.Wait()
}
```

Congratulation, running this code will register a new device to the Mir Server and your journey begins 🚀.

In a seperation terminal, run `mir device list` to see your online device.

Each device is represented in the system by it's Digital Twin, use `mir device list weather_dev/default -o yaml` to see yours:

```yaml
apiVersion: v1alpha
apiName: device
meta:
    name: weather_dev
    namespace: default
    labels: {}
    annotations: {}
spec:
    deviceId: weather_dev
    disabled: false
properties: {}
status:
    online: false
    lastHearthbeat: 2024-11-15T20:01:19.296494766Z
    schema:
        packageNames:
            - google.protobuf
            - mir.device.v1
        lastSchemaFetch: 2024-11-15T20:00:03.604338288Z
```

- **Name**: The device arbritary name, this can be renamed at any time to be more friendly.
- **Namespace**: To organize devices in different groups.
- **Labels**: KeyValue pairs. To add identifying data to the device. Indexed by the system.
- **Annotations**: KeyValue pairs. To add metadata to the device. Not indexed by the system.
- **DeviceId**: The unique identifier of the device. This is the only required field.
- **Disabled**: The device will not be able to communicate with the server.
- **Properties**: Used to configure desired an reported properties of the device.
- **Status**: System information about the status of the device.

*! Note: Name and Namespace form a composable unique key while deviceId is unique.*

*! Note: When creating a device by the SDK, the device is automatically set in the `default` namespace and use deviceId for name.*

You can use `mir device edit weather_dev/default` to interactively edit the device twin. Rename it, change its namespace, add labels, etc. The CLI offers many commands to interact with devices. Yours to explore 🛰️.


### Device Communication

The SDK provices a set of function to interact with the Mir server. There are 3 types of communication:

1. **Device Telemetry (hot path)**: data are sent from the device to the server as fire and forget.
2. **Device Commands (warm path)**: data are sent from the server to the device with a reply expected.
3. **Device Configuration (cold path)**: data is exchange between the server and the device in an asynchronous way. Used to configure the device and report the current status.

To provice a great developper experience and high performance, Mir utilizes [Protocol Buffer](https://protobuf.dev/) to define the communication schema.

> Protocol buffers (protobuf) are language-agnostic, extensible mechanism for serializing structured data.
> Think of it as XML/JSON, but smaller, faster, and generates native language bindings.
> You define how you want your data to be structured once, then you can use special generated source code to easily write and read your structured data to and from a variety of data streams and using a variety of languages.

On top of Protobuf, Mir provide a predefined schema to annotate Protobuf messages with metadata to help the server understand the type of data.

For this next part, we will define a Mir protobuf schema to define communication between your device and the server.

First of, install Protobuf compiler:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Create the following file structure in your project:
```bash
.
├── go.mod
├── go.sum
├── cmds
│   └── main.go
└── schemav1
    └── schema.proto
```

You now need to import the Mir meta proto in your project to access definitions. From the root of your project, run the command:

```bash
mir tools generate mir-schema -p schemav1/
```

You will now have this file structure:

```bash
.
├── go.mod
├── go.sum
├── cmds
│   └── main.go
└── schemav1
    ├── mir
    │   └── device
    │       └── v1
    │           └── mir.proto
    └── schema.proto
```

- `mir.proto`: contains Mir metadata. This file should be copy pasted from the [Mir Repository](https://github.com/MaxThom/mir/blob/main/pkgs/device/proto/mir/device/v1/mir.proto) or generated from the CLI. It is important to respect the file structure for this one
- `schema.proto`: contains your device schema. This file and more should be created by you.

schema.proto
```proto
syntax = "proto3";

package schemav1; // Should be the same as the folder name
option go_package = "github.com/<user|org/<project>/schemav1"; // Should be the path where the generated code will be stored


import "mir/device/v1/mir.proto";
```

When you are ready, you can generate the Go code with the following command from the root of your project:
```bash
protoc --proto_path=schemav1/ \
       --go_out=schemav1 \
       --go_opt=paths=source_relative \
       schemav1/schema.proto
  ```

You should have a successful generation except a warning saying `mir.proto` is unused.
Everytime you add to the schema, you will need to regenerate the code.

Back in your `main.go`, you can now tell the SDK to use your schema in the builder pattern as well as using the generated code:
```go
package main

import (
	"context"
	"mir-device/schemav1"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxthom/mir/pkgs/device/mir"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather_dev").
		Target("nats://127.0.0.1:4222").
		Schema(schemav1.File_schema_proto). // Import the go package and pass the defined schema to Mir
		Build()
	if err != nil {
		panic(err)
	}

	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal
```

Protobuf needs a bit of getting used to, but it is a powerful tool. The generated code will help you to interact with the server in a type safe way as well, give high performance and provides a great developper experience.

You will used the code generation command often so it is a good idea to add it to a Makefile, task or a script.

From this point on, everything is setup to start building!

#### Device Telemetry

Device telemetry is the most common way to send data from the device to the server.
This is the hot path and is used to send data that does not require a reply.
This type of data is of timeseries as each datapoint sent is attached to a timestamp of different precision (you choose on your needs).
The Mir telemetry module will ingest and store it in [InfluxDB](https://www.influxdata.com):

> InfluxDB is a time series database designed to handle high write and query loads.
> InfluxDB is meant to be used as a backing store for any use case involving large amounts of timestamped data, including DevOps monitoring, application metrics, IoT sensor data, and real-time analytics.

First, lets define a telemetry message in your schema:
```proto
syntax = "proto3";

package schemav1;
option go_package = "github.com/maxthom/mir-device/schemav1";

import "mir/device/v1/mir.proto";

message EnvTlm {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELEMETRY;

	int64 ts = 1 [(mir.device.v1.timestamp) = TIMESTAMP_TYPE_NANO];
	int32 temperature = 2;
	int32 pressure = 3;
	int32 humidity = 4;
}
```

Here we define a message `EnvTlm` that will be used. The options are used to annotate the message with metadata:

- `mir.device.v1.message_type`: This tell the server that this message is of telemetry type.
- `mir.device.v1.timestamp`: This tell the server that the field `ts` is the main timestamp and the precision is NANOSECONDS. Second, Microsecond and Millisecond are also available.

Lets regenerate the schema:

```bash
protoc --proto_path=schemav1/ \
       --go_out=schemav1 \
       --go_opt=paths=source_relative \
       schemav1/schema.proto
```

Let's create a function that send telemetry data to the server every 3 seconds.
To do so, we use the `m.SendTelemetry` function that take any proto message:

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

  // Start go routine for not to block main thread
	go func() {
		for {
			select {
			case <-ctx.Done():
			  // If context get cancelled, stop sending telemetry and
				// decrease the wait group for graceful shutdown
				wg.Done()
				return
			case <-time.After(3 * time.Second):
			  log.Default().Println("sending telemetry data...")
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

And just like that, we now have telemetry that his stored server side
```bash
mir tlm list weather_dev

1. weather_dev/default
schemav1.EnvTlm{} localhost:3000/explore
```

Click on the link to open Grafana and visualize the data.

Voila! You have successfully sent telemetry data to the server. Add more message to the schema and send more data!
Use the CLI to quickly get link to the telemetry data in Grafana and use the generated query to create powerful dashboard in Grafana.

*! Note: All [protobuf definition](https://protobuf.dev/programming-guides/proto3/) are supported except OneOf*

#### Device Commands

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
 	func(c proto.Message) (proto.Message, error) { // Handler function
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
mir command list weather_dev
  schemav1.ChangeDataRate{}

# If you don't see the change in your schema, add the force refresh flag.
# Available on all commands
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

# You can also use pipes to pass payload
mir cmd send weather_dev/default -n schemav1.ChangeDataRate -j > ChangeDataRate.json
# Edit ChangeDataRate.json
# Send it!
cat ChangeDataRate.json | mir cmd send weather_dev/default -n schemav1.ChangeDataRate
```

Voila! You have successfully sent a command to the device to change it's data rate.
Look at your device logs to see the data rate change. The CLI offers many more options to interact with the server, devices, do testing and validation, etc.


#### Device Configuration

Under construction. Coming Q1 2025!

## Device Template for new project (or importing)


## Ecosystem

The MIR ecosystem is composed of multiple components that work together to provide a complete solution for managing devices, communication, updates and their data.
Visit the rest of the documentation to see more detailed example and all possibilities.
