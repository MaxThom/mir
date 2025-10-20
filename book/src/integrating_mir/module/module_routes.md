# Routes

The Module SDK provides three main route types for interacting with the Mir ecosystem. Each route type serves a different purpose and follows consistent patterns.

## Route Types Overview

| Route Type | Purpose | Direction | Pattern |
|------------|---------|-----------|---------|
| **Device** | Direct device communication | Device ↔ Module | Subscribe to device streams |
| **Client** | Server-side operations | Module ↔ Mir Services | Request/Response & Subscribe |
| **Event** | System event notifications | Mir → Module | Subscribe to events & Publish |

### Common Patterns

#### Subscribe vs QueueSubscribe

All routes support two subscription modes:

- **Subscribe**: Every module instance receives all messages
- **QueueSubscribe**: Messages are distributed across module instances (worker pattern)

```go
// Standard subscription - all instances receive messages
m.Device().Telemetry().Subscribe("*", handler)

// Queue subscription - only one instance receives each message
m.Device().Telemetry().QueueSubscribe("workers", "*", handler)
```

Use `*` to subscribe to all devices or `deviceId` for a single specific device.

#### Message Acknowledgment

Always acknowledge messages after processing:

```go
func handler(msg *mir.Msg, deviceId string, data []byte) {
    // Process the message
    processData(data)

    // Acknowledge
    msg.Ack()
}
```

---

## Routes Layout

All routes in Mir follow a structured subject pattern based on NATS. Understanding this structure helps you create custom routes and understand how messages are routed.

### Route Subject Structure

Routes are composed of multiple segments separated by dots (`.`):

```
<type>.<id>.<module>.<version>.<function>.<extra...>
```

**Segment Definitions:**

1. **Type** - The route category:
   - `device` - Direct device communication
   - `client` - Server-side operations (module-to-module or module-to-service)
   - `event` - System event notifications

2. **ID** - The identifier for routing:
   - For **device** routes: `deviceId` (or `*` for all devices)
   - For **client** routes: `clientId` (or `*` for all clients)
   - For **event** routes: `eventId` (or `*` for all events)

3. **Module** - Your module/application name:
   - Identifies which module or service the route belongs to
   - Example: `"myapp"`, `"cfg"`, `"core"`, `"tlm"`, etc

4. **Version** - Schema/API version:
   - Semantic versioning: `"v1"`, `"v2"`, `"v1alpha"`
   - Allows route evolution without breaking existing subscribers

5. **Function** - The specific operation or data type:
   - What the route does or what type of data it carries
   - Example: `"list"`, `"send"`, `"update"`

6. **Extra** - Optional additional routing tokens:
   - Further refine routing as needed
   - Example: `"high-priority"`, `"zone-a"`

#### Creating Custom Routes

Use the `NewSubject()` function to create custom routes:

```go
// Device custom route
subject := m.Device().NewSubject("myapp", "v1", "temperature", "celsius")
m.Device().Subscribe(subject, handler)
// Subscribes to: device.*.myapp.v1.temperature.celsius

// Client custom route
subject := m.Client().NewSubject("myapp", "v1", "process-data")
m.Client().Subscribe(subject, handler)
// Subscribes to: client.*.myapp.v1.process-data

// Event custom route
subject := m.Event().NewSubject("*", "myapp", "v1", "alert")
m.Event().SubscribeSubject(subject, handler)
// Subscribes to: event.*.myapp.v1.alert
```

#### Wildcards

Use `*` to subscribe to multiple routes:

```go
// Subscribe to all devices
m.Device().Telemetry().Subscribe("*", handler)
// Matches: device.*.mir.v1.telemetry

// Subscribe to all events from your module
subject := m.Event().NewSubject("*", "myapp", "v1", "*")
m.Event().SubscribeSubject(subject, handler)
// Matches: event.*.myapp.v1.*
```

## Message Metadata

All routes provide access to message metadata through the `*mir.Msg` type:

```go
func handler(msg *mir.Msg, ...) {
    // Get trigger chain (array of all services that handled this message)
    chain := msg.GetTriggerChain()
    fmt.Printf("Trigger chain: %v\n", chain)

    // Get origin (first service in chain)
    origin := msg.GetOrigin()
    fmt.Printf("Origin: %s\n", origin)

    // Get original trigger ID
    originalTrigger := msg.GetOriginalTriggerId()

    // Get timestamp
    timestamp := msg.GetTime()
    fmt.Printf("Time: %s\n", timestamp)

    // Get protobuf message name (for telemetry, commands, etc.)
    msgName := msg.GetProtoMsgName()
    fmt.Printf("Proto message: %s\n", msgName)

    // Access underlying NATS message
    natsMsg := msg.Msg
    fmt.Printf("Subject: %s\n", natsMsg.Subject)
    fmt.Printf("Reply: %s\n", natsMsg.Reply)
}
```

---
