# Integrating with Mir

> **Build powerful IoT solutions with Mir's flexible SDKs** 🛠️

Mir provides two powerful SDKs that enable you to build complete IoT solutions: the **DeviceSDK** for connecting hardware devices, and the **ModuleSDK** for extending server-side capabilities. Together, they form a comprehensive platform for IoT development.

## 🎯 Choose Your Integration Path

### 🔌 **DeviceSDK** - Connect Your Hardware or Software
Build reliable, scalable device integrations with minimal code. Perfect for:
- IoT device manufacturers
- Software configuration
- Embedded systems developers
- Hardware engineers
- Sensor and actuator integration

### 🚀 **ModuleSDK** - Extend the Platform
Add custom business logic and integrations on the server side. Ideal for:
- Integrate with own ERP or databases
- Extend the system with your needs
- Custom workflow builders

---

## 🔌 DeviceSDK - Your Device Connection Layer

The DeviceSDK is your gateway to seamlessly connecting IoT devices or software with the Mir platform. It handles all the complexities of device-to-cloud communication, letting you focus on your device's or software core functionality.

### Why DeviceSDK?

**Language Independence**
- Built on Protocol Buffers for cross-language support
- Currently available for Go
- Python and C/C++ support coming soon
- Clean, idiomatic APIs for each language

**Production-Ready Features**
- ✅ Automatic reconnection and failover
- ✅ Offline data buffering
- ✅ End-to-end encryption
- ✅ Schema validation
- ✅ Built-in health monitoring

**Developer Experience**
- Simple, intuitive APIs
- Comprehensive documentation
- Example implementations
- Active community support

---

## 🚀 ModuleSDK - Your Server Extension Framework

The ModuleSDK empowers you to extend Mir's server-side capabilities, enabling powerful integrations and custom business logic that runs alongside the core platform.

### Why ModuleSDK?

**Seamless Integration**
- Direct access to all Mir services
- Event-driven architecture
- Real-time data processing

**Enterprise Ready**
- ✅ High-performance event streaming
- ✅ Transactional guarantees
- ✅ Horizontal scalability
- ✅ Built-in observability

**Flexibility**
- Build any custom logic
- Integrate with any system or your databases
- Process data your way
- Deploy independently

### 📋 Capabilities

#### **1. Event Subscriptions**
- Device lifecycle events (connect/disconnect)
- Telemetry data streams
- Command execution results
- Configuration changes
- System health updates

#### **2. Device Operations**
- Send commands to any device
- Update device configurations
- Query device states
- Manage device metadata

#### **3. External Integrations**
- Database connections
- Third-party APIs
- Message queues
- Cloud services

### 🎯 Common Use Cases

**Business Logic**
- Automated device control based on conditions
- Cross-device coordination
- Threshold monitoring and alerting
- Predictive maintenance

**Enterprise Integration**
- ERP system synchronization
- CRM data enrichment
- Billing and usage tracking
- Compliance reporting

**Analytics & ML**
- Real-time anomaly detection
- Pattern recognition
- Predictive analytics
- Custom dashboards

**Workflow Automation**
- Multi-step device operations
- Scheduled tasks
- Event-driven workflows
- Custom approval processes

---

## 🤝 Better Together

The true power of Mir emerges when you combine both SDKs:

1. **DeviceSDK** collects and transmits data from your hardware or sofware
2. **ModuleSDK** processes data and configuration and implements business logic
3. Together, they create complete end-to-end IoT solutions

---

## 🚦 Next Steps

Ready to start building? Choose your path:

### 🔌 **For Device Developers**
→ Jump into the [Device SDK Guide](./device/device_sdk.md) to connect your first device

### 🚀 **For Backend Developers**
→ Explore the [Module SDK Guide](./module/module_sdk.md) to build your first extension

---

Welcome to the Mir developer community! Let's build the future of IoT together. 🌟
