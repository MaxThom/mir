# Overview

> **Mir Server is a unified IoT platform that provides everything you need to connect, manage, and monitor devices at scale.**

Built with a focus on developer experience and production reliability, Mir handles the complex infrastructure so you can focus on your devices and data.

## 🏗️ Loosely Coupled Architecture

### The Mir Server
At its heart, Mir Server is a single, powerful application that orchestrates all IoT operations:

- **Unified Gateway**: Single entry point for all device connections
- **Protocol Management**: Handles telemetry, commands, and configuration through optimized channels
- **Digital Twin Engine**: Maintains virtual representations of all physical devices
- **Storage Orchestration**: Manages time-series data, device metadata, and configurations
- **Real-time Processing**: Streams data with sub-millisecond latency

Built as a loosely coupled architecture, each components can scale individually or run as a single unit for easy development and transportability.

## 📱 Uniform Device Management at Scale

Mir provides a consistent, unified approach to managing thousands of devices, regardless of their type or manufacturer:

### Device Organization
- **Namespaces**: Logical isolation for multi-tenant deployments or organizational units
- **Labels & Annotations**: Flexible tagging system for device categorization
- **Dynamic Groups**: Query-based device selection for bulk operations

### Fleet Management Capabilities
- **Bulk Operations**: Execute commands across thousands of devices simultaneously
- **Device Templates**: Standardized configurations for device types
- **Heterogeneous Support**: Mix sensors, actuators, gateways in one platform

### Management Examples
```bash
# Send command to all devices in production namespace
mir cmd send */production -n start_bootup

# Update configuration for all temperature sensors
mir cfg send --label="type=temp-sensor" -n datarate -p '{"interval": 60}'

# Query devices by multiple criteria
mir device list --namespace=factory --label="location=floor-2,status=online"
```

### Scaling Patterns
- **Sharding by Namespace**: Distribute load across clusters
- **Regional Deployment**: Devices connect to nearest Mir instance
- **Federation**: Link multiple Mir deployments for global scale
- **Load Balancing**: Automatic device distribution across servers

## 📊 Events & Audit Trail

Mir provides comprehensive event tracking and auditing for compliance, debugging, and operational insights:

### Event System
- **Automatic Capture**: Every device interaction generates an event
- **Rich Context**: Events include device ID, timestamp, user, action, and outcome
- **Real-time Streaming**: Subscribe to events as they happen
- **Persistent Storage**: Long-term retention in database

### Audit Capabilities
- **Complete History**: Full timeline of device lifecycle
- **Compliance Ready**: Meet regulatory requirements
- **Forensic Analysis**: Investigate issues with detailed logs
- **Custom Retention**: Configure retention per event type

### Integration Options
- **Webhook Notifications**: Push events to external systems
- **SIEM Integration**: Forward to security platforms
- **Custom Processors**: Build event-driven workflows
- **Grafana Dashboards**: Visualize event patterns

## 🔐 Protocol Buffers & Schema Exchange

### Schema-First Design
Mir uses Protocol Buffers (protobuf) as its foundation for all communication:

```protobuf
// Device defines its capabilities through schemas

message EnvironmentTlm {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELEMETRY;

	mir.device.v1.Timestamp ts = 1 [(mir.device.v1.timestamp) = TIMESTAMP_TYPE_NANO];
	int32 temperature = 2;
	int32 pressure = 3;
	int32 humidity = 4;
	int32 wind_speed = 5;
}

message ActivateHVAC {
	option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECOMMAND;

	int32 duration_sec = 1;
}

message ActivateHVACResp {
  bool success = 1;
}

message DataRateProp {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECONFIG;

  int32 sec = 1;
}

message DataRateStatus {
  int32 sec = 1;
}
```

### Dynamic Schema Exchange
When devices connect, they share their protobuf schemas with Mir Server:

1. **Device Registration**: Device sends its schema definitions
2. **Schema Validation**: Mir validates and stores the schemas
3. **API Generation**: Automatically creates type-safe APIs
3. **Dashboard Generation**: Automatically creates visualization for data
4. **Documentation**: Self-documents all device capabilities
5. **Version Management**: Handles schema evolution gracefully

Benefits:
- **Type Safety**: Compile-time validation prevents runtime errors
- **Self-Documenting**: Device capabilities are always clear
- **Language Agnostic**: Generate SDKs for any language
- **Efficient**: Binary encoding reduces bandwidth usage

## 📡 Offline and Local Capabilities

Mir is designed for real-world IoT deployments where connectivity isn't guaranteed:

### Device-Side Features
- **Local Storage**: Devices buffer data during disconnections
- **Automatic Retry**: Seamless reconnection when network returns
- **Data Prioritization**: Critical data sent first upon reconnection
- **Conflict Resolution**: Handles concurrent offline changes

### Server-Side Support
- **Digital Twins**: Device state persists even when offline
- **Command Queuing**: Commands wait for device reconnection
- **Event Sourcing**: Complete history of all device interactions
- **Flexible Sync**: Devices can sync at their own pace

### Offline Patterns
```go
// Device SDK handles offline automatically
device.SendTelemetry(data) // Buffered locally if offline

// Server store config in digital twin and device reconcile on connect
mir.Server().SendConfig().Request(deviceID, properties) // Delivered when device reconnects
```

## 🛠️ Developer SDKs

### DeviceSDK - Build Connected Devices

The DeviceSDK provides everything you need to integrate your hardware with Mir:

**Key Features**
- **Builder Pattern**: Simple, fluent API for device creation
- **Automatic Reconnection**: Built-in resilience for unreliable networks
- **Offline Buffering**: Local storage when disconnected
- **Schema Validation**: Type-safe communication via protobuf
- **Multi-Language Support**: Currently Go, with Python and C++ coming soon

**Device Capabilities**
- Stream high-frequency telemetry data
- Respond to real-time commands
- Manage configuration with digital twin sync
- Persistent local storage for reliability
- Built-in health monitoring and metrics

### ModuleSDK - Extend Server Capabilities

The ModuleSDK allows you to build custom server-side logic and integrations:

**Use Cases**
- **Custom Business Logic**: Process data, trigger alerts, automate workflows
- **Third-Party Integrations**: Connect to external APIs, services and databases
- **Data Processing**: Transform, aggregate, or enrich device data
- **Custom APIs**: Expose specialized endpoints for your applications
- **Advanced Analytics**: Build complex event processing pipelines

**Module Features**
- Access to all Mir services (Core, Telemetry, Commands, Config, Events)
- Event-driven architecture with subscriptions
- HTTP API extension capabilities
- Automatic reconnection and error handling
- Full access to device schemas and metadata
- Built-in observability with metrics and logging

**Integration Patterns**
- Subscribe to device lifecycle events
- Process telemetry streams in real-time
- Trigger actions based on device state changes
- Create custom dashboards and visualizations
- Implement complex authorization rules

## 📈 Monitoring & Observability

Mir provides comprehensive monitoring capabilities out of the box:

### Built-in Dashboards
- **Grafana Integration**: Pre-configured dashboards for all telemetry data
- **Auto-Generated Views**: Dashboards created automatically from device schemas
- **Real-time Visualization**: Live data streaming with customizable refresh rates
- **Multi-Device Views**: Compare data across device fleets

### Metrics & Health
- **Prometheus Metrics**: All services expose metrics endpoints
- **Device Health Tracking**: Monitor connection status, data rates, and errors
- **System Performance**: Track CPU, memory, network, and storage usage
- **Custom Metrics**: Add your own metrics via DeviceSDK or ModuleSDK

### Alerting Capabilities
- **Threshold Alerts**: Set limits on any telemetry value
- **Anomaly Detection**: Identify unusual patterns in device behavior
- **Connectivity Alerts**: Get notified when devices go offline
- **Integration Ready**: Connect to PagerDuty, Slack, email, and more

## 🔒 Security Features

### Device Security
- **Mutual TLS**: Certificate-based authentication
- **API Keys**: Token-based authentication option
- **Device Identity**: Unique device fingerprinting
- **Secure Enrollment**: Zero-touch provisioning

### Communication Security
- **End-to-End Encryption**: TLS 1.3 for all connections
- **Message Signing**: Prevent tampering
- **Perfect Forward Secrecy**: Protect past communications
- **Certificate Rotation**: Automatic renewal

## 🚀 Deployment Options

### Local Development
```bash
# Start supporting infrastructure
mir infra up

# Start server
mir serve
```

### Production Deployments

**Container Deployment**
- Docker image ready
- Docker Compose for server and supporting infrastructure

**Cloud Native**
- Kubernetes-ready with Helm charts
- Auto-scaling based on load
  - Multi-region support thanks to NatsIO

## 🎯 Next Steps

Now that you understand Mir's architecture:

- **Get Started**: Follow the [Quick Start](./quick_start.md) guide
- **Build Devices**: Learn the [Device SDK](./integrating_mir/device/device_sdk.md)
- **Deploy**: Check the [Deployment Guide](./running_mir/running_mir.md)
- **Operate**: Read the [Operator's Guide](./operating_mir/operating_mir.md)
