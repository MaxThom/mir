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

### Telemetry

```yaml
swarm:
  logLevel: "info"                  # Log level

  devices:
    - count: 100                      # Total devices to simulate
      meta:
        name: "device"            # ID template pattern
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
        expr: "sin(t)"            # Generator expression
        # ... generator parameters
    - name: "environmental"
      type: "message"
      tags: {}
      fields:
        - "name"
```

### Commands

Commands gonna reply the same proto message with the same data.
There is gonna be some options to add delays and else.

Another idea, is to have also an expression that transform the data. x could be
the received data for the field

```yaml
swarm:
  logLevel: "info"                  # Log level

  devices:
    - count: 100                      # Total devices to simulate
      meta:
        name: "device"            # ID template pattern
        labels: {}                    # key value pair of string, string
        annotations: {}               # key value pair of string, string
        namespace: "default"          # Default namespace
      commands:                       # Commands groups to include, each group is a proto message
        - name: "group_name"          # Proto message name
          delay: 2s                   # Delay in the reply
          fields:
            - "sensor_name"            # List of fields
            - "nested"

  fields:                  # Command field definitions
    - name: "sensor_name"
      type: "float32"             # int8|int16|int32|int64|float32|float64|message|string|bool
      tags:                       # key value pair of string, string
        - unit: "C"               # Sensor unit
    - name: "nested"
      type: "message"
      tags: {}
      fields:
        - "sensor_name"
```
