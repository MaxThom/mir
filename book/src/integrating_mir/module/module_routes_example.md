# Routes Examples

## Device Routes

**Purpose:** Subscribe to real-time data streams directly from devices.

Device routes provide direct access to device communication streams, allowing you to monitor and interact with devices in real-time.

### Pattern: Subscribe to Device Streams

All device routes follow this pattern:
1. Filter by device ID (use `"*"` for all devices or specific device ID)
2. Receive device data in handler
3. Process and acknowledge messages

### Example: Heartbeat Route

Heartbeat routes allow you to monitor device connectivity by subscribing to periodic heartbeat messages sent by devices.

```go
package main

import (
    "fmt"
    "time"
    "github.com/maxthom/mir/pkgs/module/mir"
)

func main() {
    m, _ := mir.Connect("heartbeat-monitor", "nats://localhost:4222")
    defer m.Disconnect()

    // Subscribe to heartbeats from all devices
    m.Device().Heartbeat().Subscribe(
        "*", // All devices
        func(msg *mir.Msg, deviceId string) {
        		...
            msg.Ack()
        },
    )

    // Subscribe to heartbeats from a specific device
    m.Device().Heartbeat().Subscribe(
        "critical-sensor-01",
        func(msg *mir.Msg, deviceId string) {
        		...
            msg.Ack()
        },
    )
}
```

#### Custom Device Routes
Create custom routes for your own device protocols:
```go
// Subscribe to custom device messages
sbj := m.Device().NewSubject("mymodule", "v1", "custom-data")
m.Device().Subscribe(sbj,
    func(msg *mir.Msg, deviceId string, data []byte) {
        // Handle custom data
        msg.Ack()
    })
```

---

## Client Routes

**Purpose:** Interact with Mir services for server-side operations and device management.

Client routes provide request/response interactions with Mir's core services. They support both:
- **Request**: Call a service and get a response
- **Subscribe**: Implement a service that responds to requests

### Example: List Devices

List operations demonstrate the simple request pattern for querying data from Mir services.

```go
package main

import (
    "fmt"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    m, _ := mir.Connect("device-query", "nats://localhost:4222")
    defer m.Disconnect()

    // List all devices in a namespace
    devices, err := m.Client().ListDevice().Request(mir_v1.DeviceTarget{Namespaces: []string{"default"}}, true)
    if err != nil {
        fmt.Printf("Failed to list devices: %v\n", err)
        return
    }

    fmt.Printf("Found %d devices in namespace 'default':\n", len(devices))
}
```

#### Custom Client Routes
Create custom routes for module-to-module communication:
```go
// Subscribe to custom requests and reply
sbj := m.Client().NewSubject("mymodule", "v1", "process-data")
m.Client().Subscribe(sbj,
    func(msg *mir.Msg, clientId string, data []byte) {
      // Process request
      result := processData(data)

      // Send response if needed
      if msg.Reply != "" {
				reply := &nats.Msg{
					Subject: msg.Reply,
					Data:    result,
				}
				_ = m.Bus.PublishMsg(reply)
      }
      msg.Ack()
    }
)
```

---

## Event Routes

**Purpose:** Subscribe to system events and publish custom events.

Event routes provide a publish/subscribe pattern for system-wide notifications. Events are emitted when important actions occur in the system.

### Pattern: Subscribe and Publish

Event routes follow a pub/sub pattern:
1. **Subscribe**: Listen for events (all or filtered by subject)
2. **Publish**: Emit custom events
3. Events include trigger chains to track origin

### Example: Device Online Events and Custom Publish

Subscribe to device online events and publish custom events.

```go
package main

import (
    "fmt"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    m, _ := mir.Connect("event-monitor", "nats://localhost:4222")
    defer m.Disconnect()

    // Subscribe to device online events
    m.Event().DeviceOnline().Subscribe(
        func(msg *mir.Msg, deviceId string, device mir_v1.Device, err error) {
            if err != nil {
                fmt.Printf("Error: %v\n", err)
                msg.Ack()
                return
            }

            fmt.Printf("Device came online: %s/%s\n",
                device.Meta.Namespace, device.Meta.Name)

            // Publish a custom event when a device comes online
            publishWelcomeEvent(m, device)

            msg.Ack()
        },
    )

    select {} // Keep running
}

func publishWelcomeEvent(m *mir.Mir, device mir_v1.Device) {
    // Create custom event subject
    eventSubject := m.Event().NewSubject(
        "welcome",      // Event ID
        "mymodule",     // Module name
        "v1",           // Version
        "device-hello", // Event type
    )

    // Create event
    event := mir_v1.EventSpec{
        Type:    mir_v1.EventTypeNormal,
        Reason:  "DeviceWelcome",
        Message: fmt.Sprintf("Welcome device %s", device.Meta.Name),
        RelatedObject: device.Object,
    }

    // Publish the event
    err := m.Event().Publish(eventSubject, event, nil)
    if err != nil {
        fmt.Printf("Failed to publish welcome event: %v\n", err)
    }
}
```
