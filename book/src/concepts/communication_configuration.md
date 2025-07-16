# ⚙️ Configuration

The properties reconcile loop is a digital twin pattern for persistent device configuration.

## Characteristics

- **Desired vs Reported**: Separate intended and actual states
- **Eventually consistent**: Changes apply when device is ready
- **Persistent**: Survives reboots and disconnections
- **Versioned**: Track all configuration changes
- **Bulk capable**: Update many devices at once

## How It Works

**Updated by operators**

```
Cloud                     Mir                     Device
  │                        │                         │
  ├── Update Desired ─────▶│                         │
  │                        ├── Store in Database     │
  │                        ├── Send Desired ────────▶│
  │                        │                         │
  │                        │                         ├── Update local database
  │                        │                         ├── Call config handlers
  │                        │                         │
  │                        │◀── Send Reported Props ─┤
  │                        ├── Update Digital Twin   │
  │◀── Confirm Sync ───────┤                         │
```

**Device bootup**

```
Device                       Mir
  │   (online)                │
  ├── 1a. Fetch Desired ─────▶│
  │                           │
  │◀── Send Desired Props ────┤
  │                           │
  ├── Update local database   │
  │                           │
  │   (offline)               │
  ├── 1b. Fetch from local    │
  │                           │
  ├── 2. Call config handlers │
```

## Implementation

### Schema Definition

```protobuf
// Desired property
message SampleRateConfig {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECONFIG;

    int32 interval_seconds = 1;
}

// Reported property
message SampleRateStatus {
    int32 interval_seconds = 1;
    google.protobuf.Timestamp last_update = 2;
}

// Reported property
message BatteryStatus {
    int32 percentage = 1;
    float voltage = 2;
    bool charging = 3;
}
```

**Device Side:**
```go
// Handle a desired property and report
m.HandleProperties(&schemav1.SampleRateConfig{}, func(msg proto.Message) {
	cmd := msg.(*schemav1.SimpleRateConfig)
  if desired.IntervalSeconds < 10 {
    return fmt.Errorf("interval too short: %d", desired.IntervalSeconds)
  }

  err := sensor.SetSampleRate(desired.IntervalSeconds)
  if err != nil {
    return err
  }

  if err := m.SendProperties(&schemav1.SampleRateStatus{
      LastUpdate: time.Now(),
      IntervalSeconds: desired.IntervalSeconds,
  }); err != nil {
			m.Logger().Error().Err(err).Msg("error sending data rate status")
	}

  return nil
}

// Report a property directly
if err := m.SendProperties(&schemav1.BatteryStatus{
    Percentage: 87,
    Voltage: 3.2,
    Charging: false,
}); err != nil {
	m.Logger().Error().Err(err).Msg("error sending battery status")
}
```

## Use Cases

- **Device settings**: Sample rates, thresholds, modes
- **Network configuration**: WiFi credentials, server URLs
- **Feature flags**: Enable/disable functionality
- **Calibration data**: Sensor offsets, scaling factors
- **Schedules**: Operating hours, maintenance windows
