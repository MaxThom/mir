# Module SDK

The Mir Module SDK enables you to build custom applications and services that integrate with the Mir IoT Hub ecosystem on the server side. Whether you're creating automation workflows, data processors, analytics services, or management tools, the Module SDK provides a complete set of APIs to interact with devices, events, and the Mir platform.

Use the Module SDK to extend the server side capabilities, to integrate with your own ecosystem.

## What is a Module?

A **module** is any application or service that connects to Mir Server Side to extend its functionality. Modules can:

- **Monitor devices** - Subscribe to device telemetry, heartbeats, status changes and more
- **Control devices** - Send commands and configurations to devices
- **Process events** - React to system events like device connections, disconnections, and state changes
- **Manage devices** - Create, update, delete, and query device metadata
- **Build integrations** - Connect Mir to external systems and services
- **Create automation** - Implement workflows and business logic
- **Analyze data** - Process telemetry streams for analytics and insights

### Communication Patterns

Modules communicate using three main patterns:

1. **Device Routes** (`m.Device()`) - Interact directly with device streams
   - Subscribe to telemetry, heartbeats, schemas, and custom device messages
   - Send custom communication with devices

2. **Client Routes** (`m.Client()`) - Interact with Mir services
   - CRUD operations on devices
   - Query telemetry and command history
   - Send commands and configurations via Mir services
   - Publish custom client messages

3. **Event Routes** (`m.Event()`) - React to system events
   - Subscribe to device lifecycle events (online, offline, created, updated, deleted)
   - Monitor command and configuration events
   - Publish custom events

## Key Features

- Connection Management with automatic reconnection
- Subscribe to specific device or all devices
- Subscribe to client request
- Create your own custom route
- Implement worker patterns with queue subscriptions
- Full support for authentication and encryption
  - JWT and nKeys
  - ServerOnly and Mutual TLS

## Use Cases

- **Device Management:** Build custom management tools and dashboards that provide specialized device management capabilities.
- **Integrating Services:** Connect Mir to external systems like databases, message queues, cloud services, or enterprise applications.
- **Automation & Orchestration:**: Implement complex workflows that respond to events and coordinate actions across multiple devices.
- **Monitoring & Alerting**: Build custom monitoring solutions that watch device telemetry and trigger alerts based on custom logic.
- **Data Processing**: Create data pipelines that process telemetry streams, aggregate data, or forward data to external systems.
- **Analytics**: Build real-time analytics services that process telemetry data and generate insights.

## Getting Started

Ready to build your first module? Continue to the [Getting Started](./module_getting_started.md) guide to create your first Mir module.

## Next Steps

- [Getting Started](./module_getting_started.md) - Create your first module
- [Routes](./module_routes.md) - Define custom NATS route handlers
- [Routes Example](./module_routes_example.md) - Explore practical examples
