# Welcome to Mir IoT Hub 🛰️

> **Build connected devices with Mir, an all battery included platform**

Imagine deploying thousands of IoT devices without worrying about message routing, data storage, or real-time monitoring. That's Mir – a battle-tested IoT platform that handles the complex infrastructure so you can focus on what matters: your devices and data.

## 🎯 Why Mir?

In the world of IoT, every project starts simple but quickly becomes complex:
- *"How do I handle millions of sensor readings per second?"*
- *"How can I remotely control devices across unreliable networks?"*
- *"How do I manage device configurations at scale?"*

**Mir answers these questions with a production-ready platform that scales from your laptop to the cloud.**

## 🚀 What Makes Mir Special?

### **1. All Batteries Included Platform**
Mir includes everything you need to run a production IoT system out of the box:
- **Storage**: Time-series database for telemetry, graph database for device metadata, and persistent key-value stores for local persistance on devices
- **UI & Visualization**: Pre-built Grafana dashboards, powerful CLI with terminal UI, and real-time data streaming views
- **Monitoring & Observability**: Built-in Prometheus metrics, health checks for all services, and comprehensive event logging
- **Developer Tools**: Local development, DeviceSDK for device development, ModuleSDK to extend server side capabilities, and virtual device simulators
- **Security**: TLS encryption and device authentication
- **Scalability**: Horizontal scaling, load balancing, and clustering support built-in

### **2. Three Paths to Device Communication**
Not all IoT data is created equal. Mir provides purpose-built channels for different needs:

- **🔥 Telemetry**: Stream sensor data at lightning speed
- **🔄 Commands**: Control devices with guaranteed delivery
- **⚙️ Configuration**: Manage device state with digital twins

### **2. Zero to Development in Minutes**
```bash
# Start infrastructure
mir infra up

# Launch server
mir serve

# Your IoT platform is ready! 🎉
```

### **3. Developer-First Experience**
- **Powerful CLI & TUI**: Manage everything from your terminal
- **Auto-Generated Dashboards**: Visualize data instantly in Grafana
- **Type-Safe SDKs**: Protocol Buffers prevent integration errors

No need to wire together multiple tools or build custom infrastructure – Mir provides a complete, integrated solution from day one.

### **5. Built on Giants**
- **NATS**: Ultra-fast messaging backbone
- **InfluxDB**: Purpose-built for time-series data
- **SurrealDB**: Graph database for device relationships
- **Grafana**: Beautiful dashboards out of the box
- **Prometheus**: System monitoring

## 🏗️ Real-World Ready

Mir powers IoT solutions across industries:

| Industry | Use Case |
|----------|----------|
| 🏭 **Manufacturing** | Monitor equipment health, predict failures, optimize production |
| 🏢 **Smart Buildings** | Control HVAC, lighting, and security from one platform |
| 🌾 **Agriculture** | Track soil conditions, automate irrigation, monitor crops |
| 🚛 **Logistics** | Track fleet location, monitor cargo conditions, optimize routes |
| ⚡ **Energy** | Monitor grid health, balance load, integrate renewables |

## 🎯 Perfect For

- **Device Developers**: Build IoT devices without backend complexity
- **System Integrators**: Unite diverse device fleets under one API
- **DevOps Teams**: Deploy and scale with confidence
- **Enterprises**: Handle millions of devices without breaking a sweat

## 📚 Your Journey Starts Here

### **New to Mir?**
→ Jump into the [Quick Start](./quick_start.md) guide and connect your first device in 5 minutes

### **Building Devices?**
→ Explore the [Device SDK](./integrating_mir/device/device_sdk.md) to integrate your hardware

### **Operating at Scale?**
→ Check the [Operator's Guide](./operating_mir/operating_mir.md) for production deployments

### **Want to Understand More?**
→ Dive into the [Architecture Overview](./overview.md) for the technical foundation

---

Welcome to the Mir community! Let's build the connected future together. 🚀
