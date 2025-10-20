# Device Configuration

This guide covers all the ways to configure a Mir device, including the builder pattern, YAML configuration files, and environment variables. Understanding these options allows you to flexibly configure devices for different deployment scenarios.

## Configuration Methods

There are three ways to configure a Mir device:

1. **Builder Pattern** - Programmatic configuration in code
2. **YAML Configuration File** - External configuration file
3. **Environment Variables** - System environment variables

These methods can be combined, with the following priority order (highest to lowest):
```
Builder Pattern > YAML Config > Environment Variables > Defaults
```

## Builder Pattern

The builder pattern provides a fluent API for configuring devices programmatically. This is the most flexible approach and is useful when configuration needs to be dynamic, computed at runtime or local development.

### Basic Usage

```go
m, err := mir.Builder().
    DeviceId("weather-sensor-01").
    Target("nats://mir.example.com:4222").
    LogLevel(mir.LogLevelInfo).
    Build()
```

> **Note:**
> - Device IDs cannot contain these reserved characters: `*`, `>`, `.`
> - DeviceID must be unique per instance, better to use configuration file for it

#### Adding Device ID Prefix

Add a prefix to your device ID for organizational purposes:

```go
// Using custom prefix
m, err := mir.Builder().
    DeviceId("sensor-01").
    DeviceIdPrefix(mir.IdPrefix{
        Prefix: "warehouse-a",
    }).
    Build()
// Result: "warehouse-a_sensor-01"

// Using hostname as prefix
m, err := mir.Builder().
    DeviceId("sensor-01").
    DeviceIdPrefix(mir.IdPrefix{
        Hostname: true,
    }).
    Build()
// Result: "myhost_sensor-01"
```

### Logging

Configure logging behavior for debugging and monitoring.

```go
m, err := mir.Builder().
    LogLevel(mir.LogLevelDebug).
    LogPretty(true). // Enable colors
    Build()
```

#### Custom Log Writers

```go
// Single custom writer
logFile, _ := os.Create("device.log")
m, err := mir.Builder().
    LogWriter(logFile).
    Build()

// Multiple writers (console + file)
m, err := mir.Builder().
    LogWriters([]io.Writer{
        os.Stdout,
        logFile,
    }).
    Build()
```

### Schema Management

Register Protocol Buffer schemas for telemetry and commands.

#### Registering Schemas

```go
import schemav1 "mydevice/proto/gen/schema/v1"

m, err := mir.Builder().
    Schema(schemav1.File_schema_v1_schema_proto).
    Build()
```

#### Multiple Schemas

```go
m, err := mir.Builder().
    Schema(
        schemav1.File_schema_v1_temperature_proto,
        schemav1.File_schema_v1_humidity_proto,
        schemav1.File_schema_v1_commands_proto,
    ).
    Build()
```

### Configuration File

Load configuration from external files.

#### Default Configuration File

```go
m, err := mir.Builder().
    DefaultConfigFile().
    Build()
```

**Default config file search order:**
1. `./device.yaml`
2. `~/.config/mir/device.yaml`
3. `/etc/mir/device.yaml`

#### Custom Configuration File

```go
// YAML file
m, err := mir.Builder().
    ConfigFile("/path/to/config.yaml", mir.ConfigFormatYAML).
    Build()

// JSON file
m, err := mir.Builder().
    ConfigFile("/path/to/config.json", mir.ConfigFormatJSON).
    Build()
```

### Local Persistence

Configure local message storage for offline operation. Mir DeviceSDK handles local persistence in case of disconnected from the server. If disconnected, all messages, telemetry and configuration is written to disk until reconnnect. Upon reconnection, all stored messages is sent to the server and local configuration is synchronized.

```go
m, err := mir.Builder().
    Store(mir.StoreOptions{
        FolderPath:       ".store/",
        RetentionLimit:   time.Hour * 168, // 1 week
        DiskSpaceLimit:   85,              // 85% max disk usage
        PersistenceType:  mir.PersistenceIfOffline,
        InMemory: false,
    }).
    Build()
```

**StoreOptions Fields:**
- `PersistenceType` (string) - Storage strategy:
  - `mir.PersistenceNoStorage` - No message storage
  - `mir.PersistenceIfOffline` - Store only when disconnected (default)
  - `mir.PersistenceAlways` - Store all messages

### Authentication & Security

Configure authentication and TLS encryption for secure communication.

#### JWT Authentication

```go
// Using default credential file locations
m, err := mir.Builder().
    DefaultUserCredentialsFile().
    Build()

// Or specify custom path
m, err := mir.Builder().
    UserCredentialsFile("/path/to/device.creds").
    Build()
```

**Default credential file search order:**
1. `./device.creds`
2. `~/.config/mir/device.creds`
3. `/etc/mir/device.creds`

#### TLS Server Verification (Server-Only TLS)

```go
// Using default RootCA locations
m, err := mir.Builder().
    DefaultRootCAFile().
    Build()

// Or specify custom path
m, err := mir.Builder().
    RootCAFile("/path/to/ca.crt").
    Build()
```

**Default RootCA file search order:**
1. `./ca.crt`
2. `~/.config/mir/ca.crt`
3. `/etc/mir/ca.crt`

#### Mutual TLS (mTLS)

```go
// Using default certificate locations
m, err := mir.Builder().
    DefaultClientCertificateFile().
    DefaultRootCAFile().
    Build()

// Or specify custom paths
m, err := mir.Builder().
    ClientCertificateFile("/path/to/tls.crt", "/path/to/tls.key").
    RootCAFile("/path/to/ca.crt").
    Build()
```

**Default client certificate search order:**

Certificate:
1. `./tls.crt`
2. `~/.config/mir/tls.crt`
3. `/etc/mir/tls.crt`

Key:
1. `./tls.key`
2. `~/.config/mir/tls.key`
3. `/etc/mir/tls.key`


## Configuration File

This is ideal for production deployments where configuration should be external to the code.

### File Locations

The device searches for configuration files in this order if using `DefaultConfigFile()` options:

1. `./device.yaml` - Current directory
2. `~/.config/mir/device.yaml` - User config directory
3. `/etc/mir/device.yaml` - System-wide config

```yaml
mir:
  # Server connection
  target: "nats://127.0.0.1:4222"

  # Authentication (optional)
  credentials: ""  # Path to JWT credentials file
  rootCA: ""       # Path to RootCA certificate
  tlsCert: ""      # Path to client TLS certificate
  tlsKey: ""       # Path to client TLS key

  # Logging
  logLevel: "info"  # [trace|debug|info|warn|error|fatal]

  # Device identity
  device:
    id: "my-device"  # Required if no idGenerator

    # Optional: Add prefix to device ID
    idPrefix:
      prefix: ""       # Custom prefix string
      hostname: false  # Use hostname as prefix
      username: false  # Use username as prefix

    noSchemaOnBoot: false  # Don't send schema on connection

  # Local message storage
  localStore:
    folderPath: ""              # Storage directory
    inMemory: false             # Use in-memory storage
    retentionLimit: "168h"      # Message retention duration
    diskSpaceLimit: 85          # Max disk usage percentage (0-99)
    persistenceType: "ifoffline" # [nostorage|ifoffline|always]

# Optional: Custom application configuration
user: {}
```

### Custom User Configuration

The `user:` section in YAML allows you to add custom application-specific configuration alongside Mir configuration.


```go
type AppConfig struct {
    Sensors []SensorConfig `yaml:"sensors"`
    Interval time.Duration `yaml:"interval"`
    ApiKey   string        `yaml:"apiKey" cfg:"secret"`
}

type SensorConfig struct {
    Type     string        `yaml:"type"`
    Unit     string        `yaml:"unit"`
    Interval time.Duration `yaml:"interval"`
    Password string        `yaml:"password" cfg:"secret"`
}
```

> **Note:** Use the field tag `cfg:"secret"` tag to mark sensitive fields. These will be excluded from logs.

```yaml
mir:
  target: "nats://localhost:4222"
  device:
    id: "weather-station"
user:
  interval: 60s
  apiKey: "secret-key"
  sensors:
    - type: "temperature"
      unit: "C"
      interval: 10s
      password: "sensor-pass"
    - type: "humidity"
      unit: "%"
      interval: 30s
      password: "humid-pass"
```

#### Building with Custom Config

```go
var appConfig AppConfig

m, err := mir.Builder().
    DefaultConfigFile().
    BuildWithExtraConfig(&appConfig)
if err != nil {
    panic(err)
}

// Now use your custom config
for _, sensor := range appConfig.Sensors {
    fmt.Printf("Sensor: %s, Unit: %s, Interval: %v\n",
        sensor.Type, sensor.Unit, sensor.Interval)
}
```

### Recommended Setup

Most configuration should be external in the configuration file. Use Default locations to load required files such as configuration, credentials and certificates.

See [Security](../../security/security.md) for credentials and certificate.

```go
var appConfig AppConfig

m, err := mir.Builder().
    DefaultConfigFile().
    DefaultUserCredentialsFile().
	DefaultClientCertificateFile().
	DefaultRootCAFile().
	Schema(schemav1.File_schema_v1_schema_proto).
    BuildWithExtraConfig(&appConfig)
if err != nil {
    panic(err)
}

// Now use your custom config
for _, sensor := range appConfig.Sensors {
    fmt.Printf("Sensor: %s, Unit: %s, Interval: %v\n",
        sensor.Type, sensor.Unit, sensor.Interval)
}
```

## Next Steps

Now that you understand device configuration, continue with:

- [Device Communication](./device_communication.md) - Learn how devices communicate with Mir
- [Security](../../security/security.md) - Learn how to secure your environment
