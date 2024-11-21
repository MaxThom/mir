# Integrating Mir

## DeviceSDK

The Device SDK is your gateway to connecting IoT devices with the Mir platform. It provides a streamlined interface that handles all the complexities of device-to-cloud communication, letting you focus on what matters - your device's core functionality.

The SDK is built on Protocol Buffers to be language-independent, with initial support for Go and planned expansions to Python, C and beyond. This means you can:

- Write device code in your preferred language
- Define clean data contracts using Protobuf schemas
- Let the SDK handle messaging, security and connectivity
- Focus on implementing your device's business logic

Whether you're building sensors, controllers, or other IoT devices, the Device SDK provides the foundation for robust and secure communication with the Mir ecosystem while keeping your code clean and maintainable.


 ## ModuleSDK

The Module SDK empowers developers to extend Mir's server-side capabilities through custom modules, enabling seamless integration with external systems and enhanced functionality.

Key Capabilities:
- Create custom server-side extensions
- Integrate with external systems (databases, ERPs, analytics platforms)
- Build automated workflows and business logic
- Implement custom data processing and reporting

The SDK provides a comprehensive API to:
- Subscribe to real-time device telemetry streams
- Send commands to devices or groups of devices
- Manage device lifecycle (creation, updates, deletion)
- Listen to system events and status changes

A powerful event system allows modules to react to system-generated notifications. For example, when a device's heartbeat indicates a status change (online/offline), the core module generates events that your custom modules can handle. This enables:

- Real-time monitoring and alerting
- Automated responses to device state changes
- Integration with external notification systems
- Custom business logic triggers

Whether you're building enterprise integrations, analytics pipelines, or custom automation workflows, the Module SDK provides the tools to extend Mir's capabilities while maintaining clean separation of concerns.

Example Use Cases:
- Sync device data with enterprise systems
- Generate automated reports and analytics
- Create custom monitoring and alerting modules
- Build specialized device management workflows
- Implement domain-specific business logic
