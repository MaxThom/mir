# Digital Twin

The Digital Twin is a fundamental concept in Mir IoT Hub that provides a virtual representation of each physical device. It serves as the single source of truth for device state, configuration, and metadata, enabling powerful management capabilities even when devices are offline.

## What is a Digital Twin?

A Digital Twin in Mir is a comprehensive virtual model that mirrors your physical device:

```yaml
apiVersion: mir/v1alpha
kind: device
meta:
    name: temperature-sensor-01
    namespace: building-a
    labels:
        type: sensor
        location: floor-1
        room: conference-room
    annotations:
        manufacturer: "Acme Sensors Inc"
        installed: "2024-01-15"
spec:
    deviceId: temp-sensor-01
    disabled: false
properties:
    desired:
        sensor.v1.SampleRate:
            intervalSeconds: 60
        sensor.v1.AlertThreshold:
            maxTemp: 25.0
            minTemp: 18.0
    reported:
        sensor.v1.SampleRate:
            intervalSeconds: 60
        sensor.v1.AlertThreshold:
            maxTemp: 25.0
            minTemp: 18.0
        sensor.v1.BatteryStatus:
            percentage: 87
            voltage: 3.2
status:
    online: true
    lastHeartbeat: 2024-11-21T10:30:45Z
    schema:
        packageNames:
            - sensor.v1
            - mir.device.v1
        lastSchemaFetch: 2024-11-21T09:00:00Z
    properties:
        desired:
            sensor.v1.SampleRate: 2024-11-20T14:00:00Z
            sensor.v1.AlertThreshold: 2024-11-19T10:00:00Z
        reported:
            sensor.v1.SampleRate: 2024-11-20T14:05:00Z
            sensor.v1.AlertThreshold: 2024-11-19T10:02:00Z
            sensor.v1.BatteryStatus: 2024-11-21T10:30:00Z
```

## Core Components

### 1. **Metadata (`meta`)**

Device identification and organization:

```yaml
meta:
    name: temperature-sensor-01      # Human-readable name
    namespace: building-a            # Logical grouping
    labels:                         # Indexed key-value pairs
        type: sensor
        location: floor-1
    annotations:                    # Non-indexed metadata
        notes: "Replaced battery on 2024-10-01"
```

**Best Practices:**
- Use descriptive names following a naming convention
- Organize devices into logical namespaces
- Use labels for filtering and searching
- Store additional context in annotations

### 2. **Specification (`spec`)**

Core device configuration:

```yaml
spec:
    deviceId: temp-sensor-01    # Unique device identifier
    disabled: false             # Enable/disable device
```

The `deviceId` is immutable and must be unique across the entire system.

### 3. **Properties**

The heart of the Digital Twin pattern - split into desired and reported states:

#### **Desired Properties**
Configuration sent from the cloud to the device:
```yaml
properties:
    desired: # Cloud edit-only property
        sensor.v1.SampleRate:
            intervalSeconds: 60
```

#### **Reported Properties**
Current state reported by the device:
```yaml
properties:
    reported: # Device edit-only property
        sensor.v1.SampleRate:
            intervalSeconds: 60
        sensor.v1.BatteryStatus:
            percentage: 87
```

### 4. **Status**

Real-time device information maintained by the system:

```yaml
status:
    online: true
    lastHeartbeat: 2024-11-21T10:30:45Z
    schema:
        packageNames: [sensor.v1]
        lastSchemaFetch: 2024-11-21T09:00:00Z
    properties: # Timestamps of last updates
        desired:
            sensor.v1.SampleRate: 2024-11-20T14:00:00Z
        reported:
            sensor.v1.BatteryStatus: 2024-11-21T10:30:00Z
```

## Property Synchronization Flow

The Digital Twin pattern enables reliable configuration management through a reconciliation process:

```
1. Users Updates Desired Property
   ┌─────────────┐                                                      ┌─────────────┐
   │    Cloud    │ ─── sensor.v1.SampleRate { intervalSeconds: 30 } ──▶ │   Device    │
   └─────────────┘                                                      └─────────────┘

2. Device Receives Update
   ┌─────────────┐                                                      ┌─────────────┐
   │    Cloud    │ ◀─── Acknowledge Receipt ─────────────────────────── │   Device    │
   └─────────────┘                                                      └─────────────┘

3. Device Applies Configuration
   ┌─────────────┐                                                      ┌─────────────┐
   │    Cloud    │                                                      │   Device    │
   └─────────────┘                                                      │  (updating) │
                                                                        └─────────────┘

4. Device Reports New State
   ┌─────────────┐                                                      ┌─────────────┐
   │    Cloud    │ ◀── sensor.v1.SampleRate { intervalSeconds: 30 } ─── │   Device    │
   └─────────────┘                                                      └─────────────┘

5. Digital Twin Synchronized
   Both desired and reported show same value = ✓ In Sync
```

## Digital Twin Benefits

### **1. Offline Management**
Update device configuration even when it's offline:

- devices store their properties locally in case of restart while offline
- changes apply when device reconnects

### **2. Consistent State**
Single source of truth for device configuration across your entire fleet.

### **3. Bulk Operations**
Update thousands of devices with a single command using label selectors.

### **4. Version Control**
Track all configuration changes with timestamps and audit trails.

### **5. Integration Ready**
External systems can interact with devices through their digital twins via APIs.

## Common Use Cases

### **Configuration Management**
```yaml
desired:
    wifi.v1.Settings:
        ssid: "IoT-Network"
        channel: 6
```

### **Firmware Updates**
```yaml
desired:
    firmware.v1.Update:
        version: "2.1.0"
        url: "https://updates.example.com/v2.1.0"
        checksum: "sha256:..."
```

### **Operational Modes**
```yaml
desired:
    device.v1.Mode:
        mode: "maintenance"
        duration: 3600
```

### **Feature Flags**
```yaml
desired:
    features.v1.Flags:
        enableAdvancedMetrics: true
        enablePredictiveMaintenance: false
```

## Next Steps

Now that you understand Digital Twins, explore:
- [Communication Patterns](./communication_patterns.md) - How devices sync with their twins
- [Device SDK](../integrating_mir/device/device_sdk.md) - Implement Digital Twin in your device
- [Configuration Guide](../integrating_mir/device/device_configuration.md) - Practical configuration examples

The Digital Twin pattern is powerful yet simple - start using it to manage your device fleet more effectively! 🚀
