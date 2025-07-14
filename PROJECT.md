# Mir Project Overview

Mir IoT Hub is a comprehensive IoT platform written in Go that enables secure communication between IoT devices and cloud services. It provides device management, telemetry collection, command execution, and configuration management through a microservices architecture.

## Architecture

The system consists of several microservices that communicate via NATS message bus:

- **Core Service** (`cmds/core/`): Central management service for devices, schemas, and system administration
- **ProtoTlm Service** (`cmds/prototlm/`): Handles telemetry data ingestion and storage to InfluxDB
- **ProtoCmd Service** (`cmds/protocmd/`): Manages command execution on devices
- **ProtoCfg Service** (`cmds/protocfg/`): Handles device configuration management
- **EventStore Service** (`cmds/eventstore/`): Stores and manages system events in SurrealDB
- **Mir CLI** (`cmds/mir/`): Command-line and TUI interface for system management. Both server and client.

Key architectural patterns:
- All services use Protocol Buffers for data serialization
- NATS is used for inter-service communication with subject-based routing
- SurrealDB stores device metadata and events
- InfluxDB stores time-series telemetry data
- Each service implements health checks and metrics endpoints

## SDKs

### Device SDK

Devices integrate using the SDK in `pkgs/device/mir/`:
- Builder pattern for device creation
- Automatic reconnection handling
- Schema validation via protobuf
- See examples in `examples/` directory

### Module SDK

For building custom modules that interact with Mir:
- SDK in `pkgs/module/mir/`
- Provides clients for all Mir services
- Handles NATS connection management

## Tech Stack

### Core Technologies
- **Language**: Go (Golang)
- **Serialization**: Protocol Buffers (protobuf) with buf for code generation
- **Message Bus**: NATS for inter-service communication

### Databases
- **SurrealDB**: Graph database for device metadata, relationships, and events
- **InfluxDB**: Time-series database for telemetry data storage

### Frameworks & Libraries
- **CLI Framework**: Kong for command-line argument parsing
- **TUI Framework**: Bubble Tea for terminal user interfaces
- **HTTP Server**: HTTP/2 with h2c (HTTP/2 over cleartext)
- **Logging**: Zerolog for structured logging
- **Configuration**: knadh/koanf for configuration management

### Development Tools
- **Task Runner**: Just command runner
- **Hot Reload**: Air for hot-reload
- **Testing**: Go testing framework with integration test support
- **Terminal Multiplexer**: tmux with tmuxifier layouts
- **Documentation**: mdBook for documentation server

### Monitoring & Observability
- **Metrics**: Prometheus-compatible metrics endpoints
- **Dashboards**: Grafana with custom dashboards
- **Health Checks**: Built-in health endpoints for all services

### Container & Deployment
- **Containerization**: Docker with Docker Compose
- **CI/CD**: GitHub Actions for testing and releases

## Development Commands

- Read the file `justfile` to learn about the development commands

## Testing Strategy

- Unit tests: Run with `go test ./...`
- Integration tests: Located in files ending with `_integration_test.go`
- To run integration tests locally:
  - Integration tests require running infrastructure (use `just infra`)
  - Integration tests require running services (use `just infra`)
  - Run `just test`

## Protocol Buffers

All API and device schemas use Protocol Buffers:
- API definitions: `pkgs/api/proto/mir_api/v1/`
- Device SDK proto: `pkgs/device/proto/mir/device/v1/`
- Generate code: `just protogen` (uses buf for generation)

## Configuration

Services use YAML configuration files with environment variable overrides:
- Config precedence: CLI flags > env vars > config file > defaults
- Environment variables use `MIR_` prefix
- Config structs are in each service's main.go
- Use boiler packages for consistent config handling

## Key Dependencies

- NATS for messaging
- SurrealDB for device/event storage
- InfluxDB for time-series telemetry
- Protocol Buffers for serialization
- Zerolog for structured logging
- Kong for CLI parsing
- Koanf for configuration
- Bubble Tea for TUI components
