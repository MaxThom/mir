# Anatomy of a Mir Device

At the core of the devices, it is the device unique identifier or `deviceId`.

It is the responsibility of the dev/user to manage those ids. In a near future, the system will provide different generation methods and helpers such as UUID, MAC address, load from files, etc.
In the mean time, it is yours to implement. We can have many deployments of the same device code as long as each device have a unique id.

In this example, the deviceId will be hardcoded to `weather`.
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
		DeviceId("weather").
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

Each device is represented in the system by it's Digital Twin, use `mir device list weather/default -o yaml` to see yours:

```yaml
apiVersion: v1alpha
apiName: device
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

You can use `mir device edit weather/default` to interactively edit the device twin.
Rename it, change its namespace, add labels, etc. Only the Meta, Spec and Properties can be edited. Status is reserved for the system or extensions.
The CLI offers many commands to interact with devices. Yours to explore 🛰️.
