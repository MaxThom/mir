# Quick Start Guide

> **Get your first IoT device connected in under 5 minutes** ⚡

Welcome! This guide will walk you through setting up Mir and connecting your first virtual device. By the end, you'll understand how to stream telemetry, send commands, and manage device configurations.

## 📋 Prerequisites

Before we begin, ensure you have:

- [Docker](https://www.docker.com/) installed and running
- Terminal or command prompt access
- 5 minutes of your time

## 🚀 Installation

### Step 1: Download Mir

Download the latest release of Mir from the [releases page](https://github.com/MaxThom/mir/releases).
From the download, extract the binary. Add it to your path for easier usage.

You can also install the binary via Go (as it is a private repository, follow the [access guide](reference/access_mir.md)):
```bash
go install github.com/maxthom/mir/cmds/mir@latest
```

### Step 2: Verify Installation

```bash
mir --version
```

**Success!** You should see the Mir version information.

## 🎮 Start Your IoT Platform

Let's bring up your personal IoT platform in two simple commands:

### Terminal 1: Infrastructure
```bash
mir infra up
```
This starts the supporting services.

### Terminal 2: Mir Server
```bash
mir serve
```
This launches the Mir server that manages all your devices.

**Congratulations!** Your IoT platform is now running!

### Access Your Interfaces

**Cockpit** — the Mir web interface for device management:
- **URL**: http://localhost:3015

**Grafana** — pre-configured dashboards for telemetry visualization:
- **URL**: http://localhost:3000
- **Username**: `admin` / **Password**: `mir-operator`

Find the full list of running services [here](running_mir/binary.md).

## 🤖 Create Your First Virtual Device

Let's create a virtual device using the Swarm to see Mir in action:

### Terminal 3: Virtual Device
```bash
mir swarm --ids power
```

This creates a virtual "power monitoring" device that simulates:
- Temperature readings
- Power consumption data
- HVAC control capabilities

## 🔍 Explore Your Device

### Terminal 4: View Connected Devices
```bash
mir device list
```

**Output:**
```
NAME/NAMESPACE    DEVICE_ID    STATUS    LAST_HEARTBEAT    LABELS
power/default     power        online    Just now
```

### Inspect Device Details
```bash
mir device ls power
```

This displays the device's digital twin - a complete virtual representation including:
- Metadata (name, namespace, labels)
- Configuration (desired and reported properties)
- Status (online state, schema info)

## 💬 Communicate with Your Device

### Telemetry - Stream Real-time Data

Device sending data to the server in a fire-and-forget manner.

View incoming sensor data:
```bash
mir dev tlm list power
```

**Output:**
```
power/default
├─ EnvironmentTlm (temperature, humidity) → View in Grafana
└─ PowerConsumption (watts, voltage) → View in Grafana
```

Click the Grafana links to see live data visualization!

### Commands - Control Your Device

Commands are sent from the server to the device as a request-reply.

See available commands:
```bash
mir dev cmd send power
```

Send a command to activate HVAC:
```bash
# See command payload
mir dev cmd send power/default -n swarm.v1.ActivateHVAC -j
# Send command with modified payload
mir dev cmd send power/default -n swarm.v1.ActivateHVAC -p '{"durationSec": 5}'
# Quickly edit and send a command
mir dev cmd send power/default -n swarm.v1.ActivateHVAC -e
```

**Output:**
```
Command sent successfully!
Response: {"success": true}
```

### Configuration - Manage Device State

Configuration is divided into desired properties and reported properties. Contrary to commands, properties use an asynchronous messaging model and are written to storage. They are meant to represent the desired and current state of the device.

View current configuration option:
```bash
mir dev cfg send power/default
```

Update device configuration:
```bash
# See current config
mir dev cfg send power/default -n swarm.v1.DataRateProp -c
# See config template payload
mir dev cfg send power/default -n swarm.v1.DataRateProp -j
# Send config with modified payload
mir dev cfg send power/default -n swarm.v1.DataRateProp -p '{"sec": 5}'
# Quickly edit and send a config
mir dev cfg send power/default -n swarm.v1.DataRateProp -e
```

The device will:
1. Receive the new desired configuration
2. Apply the changes
3. Report back the updated state

## 🎯 What Just Happened?

In just a few minutes, you've:

- ✅ **Deployed** a complete IoT platform
- ✅ **Connected** a virtual device
- ✅ **Streamed** real-time telemetry data
- ✅ **Sent** commands to control the device
- ✅ **Updated** device configuration
- ✅ **Managed** your fleet via Cockpit at `localhost:3015`
- ✅ **Visualized** data in Grafana dashboards at `localhost:3000`

## 🧹 Clean Up

When you're done experimenting:

```bash
# Stop the virtual device (Ctrl+C in Terminal 3)
# Stop Mir server (Ctrl+C in Terminal 2)
# Stop infrastructure
mir infra down
```

## 🚦 Next Steps

Now that you've experienced Mir's capabilities:

### 🔧 **Build Real Devices**
→ Learn how to integrate your hardware with the [Device SDK](./integrating_mir/device/device_sdk.md)

### 📈 **Scale Your Deployment**
→ Deploy Mir in production with our [Deployment Guide](./running_mir/running_mir.md)

### 🎨 **Customize Your Platform**
→ Extend Mir with custom modules using the [Module SDK](./integrating_mir/module/module_sdk.md)

### 💡 **Explore Advanced Features**
→ Master the CLI with our [Complete CLI Guide](./operating_mir/mir_cli_tui.md)

---

🎉 **Welcome to the Mir community!** You're now ready to build amazing IoT solutions. If you have questions, check our [FAQ](./resources/faq.md) or [join our community](https://github.com/MaxThom/mir/discussions).
