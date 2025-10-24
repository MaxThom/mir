# Mir Swarm

Mir Swarm is a powerful device simulator that enables you to create and manage virtual IoT device fleets for testing, development, and demonstrations. It can simulate hundreds or thousands of devices with realistic sensor patterns, all defined through simple YAML configuration files.

## Overview

The Swarm feature provides a flexible way to:
- **Performance Test** your Mir deployment under realistic device loads
- **Load Test** the system with concurrent devices sending telemetry
- **Develop Features** using test data without physical hardware
- **Demo Capabilities** with realistic IoT scenarios
- **CI/CD Testing** with reproducible device swarms

## Quick Start

### Simple Device Swarm

The quickest way to start a swarm is with device IDs:

```bash
# Start a swarm with specific device IDs
mir swarm --ids=reco,capstone,pathstone
```

This creates simple devices that send basic telemetry data.

### Advanced Swarm with Configuration

For more complex scenarios, use a YAML configuration file:

```bash
# Generate a template configuration
mir swarm -j > my-swarm.yaml

# Edit the file to customize your swarm
# Launch the swarm from file
mir swarm -f my-swarm.yaml

# Or pipe the configuration
cat my-swarm.yaml | mir swarm

# Or run the example swarm
mir swarm -j | mir swarm
```

## Configuration Structure

```yaml
apiVersion: mir/v1alpha
kind: swarm
meta:
  name: "local"           # Schema package name component
  namespace: "swarm"      # Schema package namespace
  labels: {}
  annotations: {}
swarm:
  logLevel: info          # debug|info|warn|error
  devices: []             # Device group definitions
  fields: []              # Field definitions (shared across devices)
```

### Device Groups

Define groups of devices with shared characteristics:

```yaml
devices:
  - count: 100                    # Number of devices to create
    meta:
      name: "sensor"              # Device name prefix (sensor__0, sensor__1, ...)
      namespace: "swarm"        # Namespace for devices
      annotations:
        swarm: "true"             # Custom annotations
        environment: "test"
      labels:
        type: "environmental"     # Custom labels
    telemetry: []                 # Telemetry message definitions
    commands: []                  # Command message definitions
    properties: []                # Configuration properties definitions
```

**Note:** If `count: 1`, the device name is used as-is. For `count > 1`, devices are named `{name}__{index}` (e.g., `sensor__0`, `sensor__1`).

### Telemetry Configuration

Define telemetry messages that devices will send periodically:

```yaml
telemetry:
  - name: Environment           # Proto message name
    interval: 5s                # Send interval (e.g., 1s, 30s, 1m)
    tags:                       # Message-level tags
      unit_system: "metric"
    fields:                     # List of field names (defined in fields section)
      - temperature
      - humidity
      - pressure
```

### Commands Configuration

Define commands that devices can handle:

```yaml
commands:
  - name: ActivateHVAC          # Proto message name
    delay: 2s                   # Response delay simulation
    tags:
      category: "control"
    fields:                     # Command parameters
      - power
      - duration
```

Commands automatically echo back the received data after the specified delay, simulating device processing time.

### Properties Configuration

Define configuration properties (desired/reported state):

```yaml
properties:
  - name: SensorConfig          # Proto message name
    delay: 1s                   # Time to apply configuration
    tags:
      type: "settings"
    fields:                     # Configuration fields
      - sampleRate
      - enabled
```

When desired properties are sent to a device, the swarm will update the reported properties with the same values after the specified delay.

## Field Definitions

Fields are the building blocks for telemetry, commands, and properties. They can be value types or nested message types.

```yaml
fields:
  - name: temperature
    type: float64               # int8|int16|int32|int64|float32|float64|message
    tags:
      unit: "C"
      sensor: "DHT22"
    generator:                  # For telemetry data generation
      expr: "20 + 5*sin(t)"     # Mathematical expression
```

Create nested message structures by composing other fields:

```yaml
fields:
  - name: environmentalData
    type: message               # Composite type
    tags:
      category: "sensors"
    fields:                     # References to other field definitions
      - temperature
      - humidity
      - pressure
  - name: consumption
    type: message
    fields:
      - power
      - energy
```

### Generator Expressions

For telemetry field only, expressions use `t` as the time variable and support mathematical functions:

```yaml
generator:
  expr: "10*sin(t) + 3"
```

Supported Functions

| Function | Description | Example |
|----------|-------------|---------|
| `sin(x)` | Sine | `10*sin(t)` |
| `cos(x)` | Cosine | `5*cos(t)` |
| `tan(x)` | Tangent | `tan(t/10)` |
| `abs(x)` | Absolute value | `abs(sin(t))` |
| `sqrt(x)` | Square root | `sqrt(abs(t))` |
| `pow(x,y)` | Power | `pow(t, 2)` |
| `exp(x)` | Exponential | `exp(t/100)` |
| `log(x)` | Natural log | `log(t+1)` |
| `log10(x)` | Base-10 log | `log10(t+1)` |
| `floor(x)` | Floor | `floor(sin(t)*10)` |
| `ceil(x)` | Ceiling | `ceil(cos(t)*10)` |
| `round(x)` | Round | `round(tan(t))` |
| `min(x,y)` | Minimum | `min(sin(t), 0.5)` |
| `max(x,y)` | Maximum | `max(cos(t), -0.5)` |
| `rand` | Random | `rand(0, 100)` |

Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `pi` or `π` | 3.14159... | Pi constant |
| `e` | 2.71828... | Euler's number |
| `t` | time.Now() | Current time |

### Example Patterns

```yaml
# Oscillating temperature (15-25°C)
generator:
  expr: "20 + 5*sin(t/60)"

# Random noise around baseline
generator:
  expr: "100 + rand*20 - 10"

# Exponential growth with cap
generator:
  expr: "min(100, exp(t/1000))"

# Square wave pattern
generator:
  expr: "floor(sin(t)) * 100"

# Dampened oscillation
generator:
  expr: "exp(-t/1000) * sin(t)"

# Combined patterns
generator:
  expr: "50 + 20*sin(t/30) + 5*cos(t/10) + rand*2"
```

## Complete Example

Here's a comprehensive example demonstrating all features:

```yaml
apiVersion: mir/v1alpha
kind: swarm
meta:
  name: "perftest"
  namespace: "testing"
  labels:
    environment: "staging"
  annotations:
    created-by: "mir-swarm"
swarm:
  logLevel: info
  devices:
    # Environmental sensor fleet
    - count: 1
      meta:
        name: env-sensor
        namespace: default
        annotations:
          location: "warehouse"
        labels:
          type: "environmental"
      telemetry:
        - name: Environment
          interval: 10s
          tags:
            unit_system: "metric"
          fields:
            - temperature
            - humidity
            - pressure
        - name: AirQuality
          interval: 30s
          fields:
            - co2
            - voc
      commands:
        - name: Calibrate
          delay: 3s
          fields:
            - calibrationMode
      properties:
        - name: SensorConfig
          delay: 1s
          fields:
            - sampleRate
            - enabled
    # Power monitoring fleet
    - count: 1
      meta:
        name: power-monitor
        namespace: default
        labels:
          type: "power"
      telemetry:
        - name: PowerMetrics
          interval: 5s
          tags:
            unit_system: "metric"
          fields:
            - consumption
      commands:
        - name: ResetMetrics
          delay: 1s
          fields:
            - resetType

  # Field definitions
  fields:
    # Environmental sensors
    - name: temperature
      type: float64
      tags:
        unit: "C"
      generator:
        expr: "22 + 3*sin(t/120)"
    - name: humidity
      type: float64
      tags:
        unit: "%"
      generator:
        expr: "60 + 15*cos(t/180) + 2"
    - name: pressure
      type: float64
      tags:
        unit: "Pa"
      generator:
        expr: "101325 + 100*sin(t/300)"
    - name: co2
      type: float64
      tags:
        unit: "ppm"
      generator:
        expr: "400 + 50*sin(t/600)"
    - name: voc
      type: float64
      tags:
        unit: "ppb"
      generator:
        expr: "100 + 30*cos(t/450)"
    # Power monitoring
    - name: consumption
      type: message
      fields:
        - power
        - energy
    - name: power
      type: float64
      tags:
        unit: "W"
      generator:
        expr: "100 + 30*cos(t/450)"
    - name: energy
      type: float64
      tags:
        unit: "kWh"
      generator:
        expr: "t/3600"
    # Command/config fields
    - name: calibrationMode
      type: int32
      tags:
        description: "Calibration mode: 0=auto, 1=manual"
    - name: sampleRate
      type: int32
      tags:
        unit: "Hz"
    - name: enabled
      type: bool
    - name: resetType
      type: string
```

```bash
mir swarm -f performance-test.yaml
```

Mir Swarm helps with testing and development by providing a declarative, scalable way to simulate device fleets. Whether you need to validate system performance with thousands of concurrent devices, develop new features without physical hardware, or create compelling demonstrations, Swarm delivers the flexibility and realism required.
