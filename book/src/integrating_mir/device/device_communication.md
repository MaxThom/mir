# Device Communication

The SDK provides a set of functions to interact with the Mir server. There are 3 types of communication:

1. **Telemetry**: data are sent from the device to the server as fire and forget.
2. **Commands**: data are sent from the server to the device with a reply expected.
3. **Configuration**: data is exchange between the server and the device in an asynchronous way. Used to configure the device and report the current status.

See [Communicaton](../../concepts/communication_patterns.md).

> To provice a great developper experience and high performance, Mir utilizes [Protocol Buffer](https://protobuf.dev/) to define the communication schema. On top of Protobuf, Mir provide a predefined schema to annotate Protobuf messages with metadata to help the server understand the type of data. See [Mir Protobuf](../../concepts/protobuf.md).

## Editing the Schema

For this next part, we will define a schema to enable communication between your device and the server.

- `mir.proto`: contains Mir metadata and Protobuf extentions. Readonly.
- `schema.proto`: contains your device schema and defines the communication interface.

```proto
syntax = "proto3";

package schema.v1;
option go_package = "github.com/maxthom/mir.device.buff/proto/schema/v1/schemav1";

import "mir/device/v1/mir.proto";
```

You can also remove the rest of the schema as we will recreate it during the tutorial.

When you are ready, generate the go code:
```sh
just proto
#or
make proto
```

You should see a new file containing the generated code: `proto/schema/v1/schema.pb.go`

## Pass the Schema to Mir

Back in your `main.go`, you must import and add the proto schema to the MirSDK:

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxthom/mir/pkgs/device/mir"
	schemav1 "github.com/maxthom/mir.device.buff/proto/gen/schema/v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := mir.Builder().
		DeviceId("weather").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		Schema(schemav1.File_schema_v1_schema_proto).
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

	cancel()
	wg.Wait()
}

```

Protobuf needs a bit of getting used to, but it is a powerful tool. The generated code will help you interact with the server in a type safe way, give high performance and provide a great developper experience.

🥳 Congratulation! You have generated your schema and Mir is now aware of it.
If you run the code again and run `mir dev ls weather -o yaml`, you should see the schema in the digital twin status section:

```yaml
schema:
  packageNames:
    - google.protobuf
    - mir.device.v1
    - schema.v1
    lastSchemaFetch: "2025-07-18T10:52:07.664623316Z"
```

From this point on, everything is setup to start building!
