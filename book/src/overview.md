# Overview

The system consists of several core components:

- **Mir Server**: Central hub that manages device connections, telemetry, commands, and configuration
- **Device SDK**: Client libraries for devices to connect and communicate with the hub
- **Module SDK**: Server libraries for extending server capabilities with custom modules
- **CLI/Web UI**: Tools for managing and monitoring the system
- **Supporting Infrastructure**:
  - NATS: High-performance message bus
  - InfluxDB: Time-series database for telemetry
  - Grafana: Data visualization and dashboards
  - Prometheus: System monitoring and alerting

### Devices

Devices are the central actors in the system. They each have a unique device ID, organized into namespaces, and are represented by a digital twin in the server.

Device communication with the Mir server happens through three main channels:

- **Telemetry (hot path)**: One-way device-to-server data streaming for telemetry, metrics, logs, and events
- **Commands (warm path)**: Two-way server-to-device request/response for remote operations
- **Configuration (cold path)**: Two-way device configuration through a digital twin pattern

The device SDK provides a clean builder pattern API to establish connection with Mir server. Device authors need only focus on their business logic while the SDK handles:

- Connection management and automatic reconnection
- Message serialization using protocol buffers
- Schema management and validation
- Telemetry buffering and batching
- Command routing and responses
- Configuration synchronization
- Visualization and monitoring

### Diagram

![Architecture Overview](logos/architecture_overview.svg)
