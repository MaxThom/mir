# Device SDK


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

## Anatomy of a Mir Device

At the core of the devices, it is the device unique identifier or `deviceId`.

It is the responsibility of the dev/user to manage those ids. In a near future, the system will provide different generation methods and helpers such as UUID, MAC address, load from files, etc.

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
	m, err := mir.Builder().
		DeviceId("<device_id>").
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

Each device is represented in the system by it's Digital Twin, use `mir device list <device_id>/default -o yaml` to see yours:

```yaml
apiVersion: v1alpha
apiName: device
meta:
    name: <device_id>
    namespace: default
    labels: {}
    annotations: {}
spec:
    deviceId: <device_id>
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

You can use `mir device edit <deviceId>/default` to interactively edit the device twin. Rename it, change its namespace, add labels, etc. The CLI offers many commands to interact with devices. Yours to explore 🛰️.


### Device Communication

The SDK provices a set of function to interact with the Mir server. There are 3 types of communication:

1. Device Telemetry (hot path): data are sent from the device to the server as fire and forget.
2. Device Commands (warm path): data are sent from the server to the device with a reply expected.
3. Device Configuration (cold path): data is exchange between the server and the device in an asynchronous way. Used to configure the device and report the current status.

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
├── main.go
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
		DeviceId("<device_id>").
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

Protobuf needs a bit of getting used to, but it is a powerful tool. The generated code will help you to interact with the server in a type safe way as well as providing a great developper experience.

You will used the code generation command often so it is a good idea to add it to a Makefile, task or a script.

From this point, everything is setup to start building!

#### Device Telemetry

#### Device Commands

#### Device Configuration
