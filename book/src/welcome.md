# Welcome to Mir Ecosystem

Mir is an IoT Hub platform designed to enable secure and reliable communication between IoT devices while providing comprehensive device management capabilities.
The platform's core features include:

- **Device Communication**
  - Two-way communication between devices and server
  - Multiple communication patterns:
    - Telemetry (hot path): Device-to-cloud data streaming
    - Commands (warm path): Cloud-to-device request/response
    - Configuration (cold path): Device twin for state management
  - Protocol buffer schemas for type safety and efficiency
- **Device Management**
  - Device registration and identity management
  - Digital twin representation of devices
  - Device organization via namespaces and labels
  - Device status monitoring and health checks
  - Support for large scale device fleets
- **Data Management**
  - Time-series telemetry storage with InfluxDB
  - Automatic dashboard generation
  - Built-in data visualization with Grafana
  - Data tagging and metadata support
  - Flexible query capabilities
- **Operations**
  - CLI tool for device management and operations
  - Web interface for system monitoring
  - Real-time device monitoring and alerts
  - Deployment automation tools
  - Extensible module system

## Key Benefits

- **Scalability**: Designed to handle large numbers of devices and high data throughput
- **Extensibility**: Modular architecture allows adding new capabilities via modules
- **Developer Experience**: Comprehensive SDKs and tools for device integration
- **Observability**: Built-in monitoring and visualization capabilities
- **Security**: Secure device authentication and communication
- **Flexibility**: Support for different deployment scenarios and integration patterns

## Target Use Cases

- IoT device fleet management
- Industrial IoT and monitoring
- Smart building automation
- Distributed sensor networks
- Remote device control and automation
- Real-time data collection and analysis

Mir aims to provide a complete solution for IoT device management while maintaining flexibility for different deployment scenarios and use cases. The platform's modular architecture and comprehensive tooling enable teams to quickly integrate and manage IoT devices at scale.

## Get started

Visit the [Quick Start](./quick_start.md) guide to set up your Mir IoT Hub instance and start connecting devices.

Visit the [DeviceSDK](./integrating_mir/device/device_sdk.md) guide to learn how to integrate your devices with Mir IoT Hub.
