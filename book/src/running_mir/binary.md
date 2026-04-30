# Binary

Mir ecosystem can be run through the Mir binary.
Head to [Github Releases](https://github.com/MaxThom/mir/releases) to download the latest version.
You will find a bundle for Linux amd64/arm64 and Windows amd64/arm64. Once downloaded, extract the files to retrieve the binary.
Add the binary to your path for easy usage.

You can also install the binary via Go (as it is a private repository, follow the [access guide](../reference/access_mir.md)):
```bash
go install github.com/maxthom/mir/cmds/mir@latest
```

## Running

Mir is composed of the Mir Server side and supporting infrastructure:

- **Mir Server**: Manage devices, ingest telemetry, send commands and configuration, etc.
- **NatsIO**: High-speed message bus.
- **SurrealDB**: Store device digital twin.
- **InfluxDB**: Store device telemetry.
- **PromStack**: Provides dashboards, alerting and monitoring of the ecosystem.

All can be run through the binary for a local setup. Mir binary act as both the client and the server providing an integrated experience.

### Supporting Infrastructure

To run the supporting infrastructure, you need `docker` and `docker compose` installed.
Mir makes it easy to have a local setup by wrapping basic `docker compose` commands:

```bash
# Start the infra
mir infra up

# Stop the infra
mir infra down

# Display running containers
mir infra ps

# Remove containers
mir infra rm

# Write docker compose to disk
mir infra print
```

All extra flags get passed to `docker compose`.
The compose files are managed under env. var `$XDG_CACHE_HOME` defaulting to `$HOME/.cache/mir/infra`.

```bash
# Cockpit
localhost:3015
# Grafana       <user>///<password>
localhost:3000 # admin///mir-operator
# InfluxDB
localhost:8086 # admin///mir-operator
# SurrealDB
localhost:8000 # root///root
# Prometheus
localhost:9090
# NatsIO
localhost:4222
```

Having embeded docker compose ensure that each distributed Mir binary can
have a easy environment as well as providing a starting point if you want to
modify the compose.

### Mir Server

Once the supporting infrastructure is up and running, open a new terminal and run:

```bash
# Run Mir Server
mir serve

# See all possible options and configuration
mir serve -h
```

### Mir Client

With both infrastructure and server started, open another terminal and you can use the CLI to interact with the system.
Use the swarm command to simulate a device connecting to the server to explore Mir ecosystem.

```bash
# Server
mir serve
# TUI
mir
# CLI
mir -h
# to interact with devices
mir device ls
# to visualize the telemetry
mir device telemetry
# to send command to devices
mir device command
# to explore and upload schemas
mir device schema
# to simulate a device connecting to the server
mir swarm
```

### Mir Cockpit

Cockpit is the Mir web interface, served by the Mir Server on port `3015`. Once `mir serve` is running, open your browser at:

```
http://localhost:3015
```

Cockpit connects to the NATS broker using the active context from your CLI config (`~/.config/mir/cli.yaml`). By default it uses the `webTarget` WebSocket URL configured in that context:

```yaml
contexts:
  - name: local
    target: nats://localhost:4222
    webTarget: ws://localhost:9222
    grafana: localhost:3000
```

To configure or switch contexts, run `mir tools config edit`.

Visit [DeviceSDK](../integrating_mir/device/device_sdk.md) documentation to integrate device.

Visit [Mir CLI](../operating_mir/mir_cli_tui.md) documentation for more information.
