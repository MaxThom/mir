# Network Resiliency

Network reliability is one of the most critical challenges in IoT systems. Devices operate in unpredictable environments where connectivity can be intermittent, databases may undergo maintenance, and network conditions constantly change. Mir IoT Hub is designed from the ground up to handle these challenges gracefully, ensuring no data is lost and systems continue operating even during disruptions.

## Why Network Resiliency Matters

IoT systems face unique connectivity challenges:

- **Device Mobility** - Devices may move through areas with poor connectivity
- **Intermittent Networks** - Cellular, Wi-Fi, and other networks can be unreliable
- **Database Maintenance** - Backend services require updates and maintenance
- **Network Partitions** - Network infrastructure can fail or become congested
- **Power Cycling** - Devices may restart unexpectedly
- **Resource Constraints** - Limited bandwidth, memory, and storage on edge devices

Without proper resiliency, these challenges lead to:
- Lost telemetry data
- Missed commands
- Inconsistent device state
- Manual intervention requirements
- System downtime

Mir addresses these challenges through a multi-layered resilience architecture that provides automatic recovery and graceful degradation.

## Three-Layer Resilience Architecture

Mir implements resilience at three complementary layers:

```
┌─────────────────────────────────────────┐
│ Device Layer                            │
│   Local Storage                         │
│   • Queues messages when offline        │
│   • Configurable persistence policies   │
│   • Local configuration                 │
└─────────────────────────────────────────┘
              ↓ ↑
┌─────────────────────────────────────────┐
│ Transport Layer                         │
│   NATS Message Bus                      │
│   • Automatic reconnection              │
│   • Connection state management         │
└─────────────────────────────────────────┘
              ↓ ↑
┌─────────────────────────────────────────┐
│ Service Layer                           │
│   Database Resilience                   │
│   • TelemetryStore in-memory buffer     │
│   • EventStore in-memory buffer         │
│   • Automatic database reconnection     │
└─────────────────────────────────────────┘
```

Each layer provides independent resilience, creating a defense-in-depth approach where failures at any level are handled gracefully.

## Device Resiliency

The Device SDK provides automatic resilience without requiring changes to your application code. When a device loses connection to the Mir server, the SDK immediately begins automatic reconnection.

While disconnected, devices continue operating normally using local storage. Mir uses an embedded key-value database, to queue messages until connectivity returns. You can choose from three persistence strategies: no storage, store only when offline (default), or always store. Storage limits protect device resources with configurable retention periods (default: 1 week) and disk space caps (default: 85% of disk).

During offline operation, your device continues collecting telemetry, handling commands from cache, and operating with its last known configuration. When connection is restored, the SDK automatically recovers all pending messages in batches, re-synchronizes configuration with the server, and resumes normal operation. This entire process is transparent to your application code—no special handling required.

In regards to device properties, the Device SDK always keep a local copy of the most up to date desired properties in case of reboot and lost of connection.

**Key takeaway:** Devices never lose data during network disruptions and automatically recover without manual intervention.

For configuration details, see [Device Local Configuration](../integrating_mir/device/device_local_configuration.md).

## Server Resiliency

When database connections are lost, services enter degraded mode rather than failing completely. Critical real-time operations like telemetry collection and device communication continue uninterrupted, while administrative functions like queries and management operations are temporarily unavailable. Once connectivity is restored, services automatically return to full functionality and recover all buffered data in the background.

**Key takeaway:** Infrastructure issues don't interrupt device communication—services gracefully degrade and automatically recover.

For monitoring and alerting setup, see [Monitoring](../operating_mir/monitoring.md).
