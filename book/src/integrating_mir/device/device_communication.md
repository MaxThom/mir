# Device Communication

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

First of, install Protobuf compiler if not done earlier:
```bash
# Mir CLI
mir tools install
# or
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

- `mir.proto`: contains Mir metadata. This file should be copy pasted from the [Mir Repository](https://github.com/MaxThom/mir/blob/main/pkgs/device/proto/mir/device/v1/mir.proto) or generated from the CLI. It is important to respect the file structure for this one.
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

  // Returns a WaitGroup for proper shutdown on context cancellation
	wg, err := m.Launch(ctx)
	if err != nil {
		panic(err)
	}

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal

  cancel()
	wg.Wait()
```

Protobuf needs a bit of getting used to, but it is a powerful tool. The generated code will help you interact with the server in a type safe way, give high performance and provides a great developper experience.

You will use the code generation command often so it is a good idea to add it to a Makefile, task or a script.

From this point on, everything is setup to start building!
