# 🔥 Telemetry

The Hot Path of the system. Designed for high-volume, one-way data streaming from devices to the cloud.

## Characteristics

- **Fire-and-forget**: No acknowledgment required
- **High throughput**: Optimized for thousands of messages per second
- **Batching**: Automatic message batching for efficiency
- **Time-series storage**: Direct pipeline to InfluxDB
- **Auto-visualization**: Automatic Grafana dashboard generation

### Offline Behaviour

If the device becomes offline, all telemetry will be stored locally until reconnection. By default, there is a retention limit of one week of the data on the device. This help prevents the device disk to be fulled and create cascading issues.

## How It Works

```
Device                    Mir                     InfluxDB            Grafana
  │                        │                         │                   │
  ├── Telemetry Batch ────▶│                         │                   │
  │   (Fire & Forget)      ├── Validate Schema       │                   │
  │                        ├── Add Tags/Metadata     │                   │
  │                        ├── Batch Write ─────────▶│                   │
  │                        │                         ├── Store ─────────▶│
  │                        │                         │                   │
```

## Implementation

### Schema Definition

```protobuf
message TemperatureTelemetry {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELEMETRY;

  int64 ts = 1 [(mir.device.v1.timestamp) = TIMESTAMP_TYPE_NANO];
  double value = 2;
  string unit = 3;
  string location = 4;
}
```

**timestamp field**

Each telemetry messages needs a timestamp field specifying the required precision.

```protobuf
TIMESTAMP_TYPE_SEC = 1; // Represents seconds of UTC time since Unix epoch (int64)
TIMESTAMP_TYPE_MICRO = 2; // Represents microseconds of UTC time since Unix epoch (int64)
TIMESTAMP_TYPE_MILLI = 3; // Represents milliseconds of UTC time since Unix epoch (int64)
TIMESTAMP_TYPE_NANO = 4; // Represents nanoseconds of UTC time since Unix epoch (int64)
TIMESTAMP_TYPE_FRACTION = 5; // Represents seconds of UTC time since Unix epoch (int64) and non-negative fractions of a second at nanosecond resolution (int32)
```

If not specified, the timestamp is applied on the server side.

**tags**

Tags can be added onto the messages or specific field and add extra information to the data in the database. Tags can be on the entire messages, or the fields.

```protobuf
message TemperatureTelemetry {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELEMETRY;

  option (mir.device.v1.meta) = {
    tags: [{
      key: "building"
      value: "A"
    }, {
      key: "floor",
      value: "4"
    }]
  };

  int64 ts = 1 [(mir.device.v1.timestamp) = TIMESTAMP_TYPE_NANO];
  int32 temperature = 2 [(mir.device.v1.field_meta) = {
    tags: [{
      key: "unit",
      value: "celcius"
    }]
  }];
}
```

### Device Side

```go
// Send telemetry - fire and forget
device.Telemetry(&temperature.TemperatureTelemetry{
    Value: 23.5,
    Unit:  "celsius",
    Location: "room-1",
})

// The SDK handles batching automatically
// No need to wait for acknowledgment
```

### See Telemetry

Using the CLI:

```bash
mir tlm list <name/namespace>
```
Open generated panel with Grafana

## Use Cases

- **High rate telemetry**
- **Sensor readings**: Temperature, humidity, pressure
- **Metrics**: CPU usage, memory, network stats
- **Events**: Motion detected, door opened
- **Logs**: Application logs, debug information
- **Location data**: GPS coordinates, signal strength
