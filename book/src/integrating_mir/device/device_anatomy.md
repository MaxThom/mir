# Anatomy of a Mir Device

At the core of a device, it is the device unique identifier or `deviceId`. It is the responsibility of the developers and operators to manage those ids as each deployments or instance must have a unique id.

To begins, lets change the deviceId from `example_device` to `weather` in the builder pattern:

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

  schemav1 "github.com/maxthom/mir.device.buff/proto/gen/schema/v1"

	"github.com/maxthom/mir/pkgs/device/mir"
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

Congratulation, running this code with `make/just run` or `go run cmd/main.go` will register a new device to the Mir Server and your journey begins 🚀.

In a seperation terminal, run `mir device list` to see your online device.

Each device is represented in the system by it's Digital Twin, use `mir device list weather/default -o yaml` to see yours:

```yaml
apiVersion: mir/v1alpha
kind: device
meta:
    name: weather
    namespace: default
    labels: {}
    annotations: {}
spec:
    deviceId: weather
    disabled: false
properties: {}
status:
    online: false
    lastHeartbeat: 2024-11-15T20:01:19.296494766Z
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

*! Note: When autoprovisionning a device, meaning the device was not created beforehand in the system, the device is automatically set in the `default` namespace and use the deviceId for name.*

You can use `mir device edit weather/default` to interactively edit the device twin.
Rename it, change its namespace, add labels, etc. Only the Meta, Spec and Properties can be edited. Status is reserved for the system or extensions.
The CLI offers many commands to interact with devices. Yours to explore 🛰️.
