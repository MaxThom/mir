# Swarm Module

## Overview

The Swarm module is a configurable device simulator for the Mir IoT Hub that enables performance testing and load testing with large numbers of virtual devices. It uses YAML configuration to define device swarms with realistic sensor data patterns.

## Key Features

- **Scalable**: Simulate hundreds or thousands of devices from a single configuration
- **Flexible Sensor Definitions**: Support for multiple data types (int8, int16, int32, int64, float32, float64)
- **Mathematical Generators**: Create realistic sensor patterns using sin, cos, tan, and other math functions
- **Sensor Groups**: Organize sensors with shared update intervals for better performance
- **Device Templates**: Define different device types with varying sensor configurations
- **Template-based IDs**: Flexible device naming patterns

## Use Cases

- **Performance Testing**: Test Mir IoT Hub under high device load
- **Load Testing**: Validate system behavior with many concurrent devices
- **Development**: Generate test data for development and debugging
- **Demo**: Create realistic IoT scenarios for demonstrations
- **CI/CD**: Automated testing with reproducible device swarms

## Configuration Structure

### Top-Level Schema

```yaml
swarm:
  logLevel: "info"                  # Log level

  devices:
    - count: 100                      # Total devices to simulate
      meta:
        idPrefix: "device"            # ID template pattern
        labels: {}                    # key value pair of string, string
        annotations: {}               # key value pair of string, string
        namespace: "default"          # Default namespace
      telemetry:                      # Telemetry groups to include, each group is a proto message with TS field
        - name: "group_name"          # Proto message name
          interval: 5s                # Update interval
          fields:
            - "field_name"            # List of telemetry fields

  telemetryFields:                # Sensor group definitions
    - name: "sensor_name"
      type: "float32"             # int8|int16|int32|int64|float32|float64|message
      tags:                       # key value pair of string, string
        - unit: "C"               # Sensor unit
      generator:                  # Value generation configuration
        type: "sin"               # Generator type
        # ... generator parameters
    - name: "environmental"
      type: "message"
      tags: {}
      fields:
        - "name"
```

### Generator Types

Generators create realistic sensor data patterns:

#### Periodic Generators (sin, cos, tan)
```yaml
generator:
  type: "sin"
  amplitude: 10.0      # Wave height/range
  frequency: 0.1       # Hz - cycles per second
  phase: 0.0           # Starting phase (degrees)
  offset: 20.0         # Base/center value
  noise: 0.5           # Random variation ±
```

#### Random Generator
```yaml
generator:
  type: "random"
  min: 1000
  max: 1020
```

#### Linear Generator
```yaml
generator:
  type: "linear"
  start: 100           # Starting value
  slope: -0.001        # Change per second
  min: 0               # Minimum bound (optional)
  max: 100             # Maximum bound (optional)
```

#### Constant Generator
```yaml
generator:
  type: "constant"
  value: 42
  noise: 0.5           # Optional random variation
```

#### Additional Generators
- **sawtooth**: Ramp wave pattern
- **square**: Square wave pattern
