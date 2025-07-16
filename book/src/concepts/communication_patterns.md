# Communication Patterns

Mir IoT Hub implements three distinct communication patterns, each optimized for specific use cases. Understanding these patterns is crucial for building efficient and reliable IoT solutions.

## The Three Paths

```
┌─────────────────────────────────────────────────────────────────┐
│                        Device Communication                     │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   🔥 Telemetry  │  🔄 Commands    │  ⚙️ Configuration           │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Fire & Forget   │ Request/Reply   │ Desired/Reported State      │
│ High Volume     │ Synchronous     │ Persistent                  │
│ One-way         │ Two-way         │ Eventually Consistent       │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

## Choosing the Right Path

| Aspect | Telemetry | Commands | Config |
|--------|---------------------|---------------------|-------------------|
| **Direction** | Device → Cloud | Cloud ↔ Device | Cloud ↔ Device |
| **Acknowledgment** | None | Required | Eventually |
| **Persistence** | Time-series DB | Event log | Digital Twin |
| **Use When** | Streaming data | Immediate action | Persistent state |
| **Offline Behavior** | Buffer locally | Fails immediately | Applies when online |
| **Examples** | Sensor data | Turn on light | Update threshold |

Remember: Choose the right path for each use case, and your IoT solution will be efficient, reliable, and scalable! 🚀

## Next Steps

Master these communication patterns to build robust IoT solutions:
- [Digital Twin](./digital_twin.md)
- [Device SDK](../integrating_mir/device/device_sdk.md)
- [Telemetry Guide](../integrating_mir/device/device_telemetry.md)
- [Commands Guide](../integrating_mir/device/device_commands.md)
- [Configuration Guide](../integrating_mir/device/device_configuration.md)
