# Local Development & Testing

## Pre-requisites

To run Mir locally, you need to have the following installed:

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://golang.org/doc/install)
- [Rust](https://www.rust-lang.org/tools/install)  (To run Mir Book)
- [Just](https://github.com/casey/just) (Optional, for common tasks)
- [Protoc](https://grpc.io/docs/protoc-installation/)
- [Npm](https://docs.npmjs.com/cli/v11/configuring-npm/install)

Once you have the above installed, you can run the following commands to complete installation:

```bash
# Linux
./scripts/tooling.sh

# Windows
go install github.com/air-verse/air@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/buf/cmd/buf@latest
cargo install mdbook@0.4.40
cargo install just
```

To finish:
```bash
git clone git@github.com:MaxThom/mir.git
```

## Running

Mir relies on a number of services to run:

- InfluxDB: A time-series database for storing telemetry data
- SurrealDB: A key-value store for storing device data
- Prometheus: A monitoring and alerting toolkit
- Grafana: A visualization tool for monitoring data
- NatsIO: A message broker for communication between device and services

These services are defined in the [docker compose](./infra/compose/local_support/compose.yaml) file. To start the services, run the following command:

```bash
just infra
# or
docker compose -f infra/compose/local_support/compose.yaml up --force-recreate
# or VsCode/Zed task
Mir infra dev
```

```bash
# Service: Grafana
# URL: http://localhost:3000
# Username: admin / Password: mir-operator

# Service: InfluxDB
# URL: http://localhost:8086
# Username: admin / Password: mir-operator

# Service: SurrealDB
# URL: http://localhost:8000
# Username: root / Password: root

# Service: Prometheus
# URL: http://localhost:9090

# Service: NATS
# URL: http://localhost:8222
```

> **Note:** Cockpit (web UI) is served by `mir serve` on `http://localhost:3015`, not by the infra stack.

To build Mir binary, run the following command:

```bash
just build
# or
go build -o bin/mir cmds/mir/main.go
```

The Mir binary comes with a powerful CLI and TUI. It acts as both the client and the server.
Once started as the server, open another terminal and you can use the CLI to interact with the system.
Use the `swarm` command to simulate a device connecting to the server to explore the Mir ecosystem.

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

**Tip:** On Linux, you can run `just install` to install the binary to your system path.

To integrate your own device to the system, visit the [device tutorial](../integrating_mir/device/device_sdk.html).

### Development

Mir is built with a module architecture. Each module is a standalone service that can be run independently or combined with the CLI.
The modules are:

- Core: handles the management of devices
- Telemetry: handles the telemetry ingestion
- Command: handles the command delivery
- Configuration: handles the configuration of devices

The repository comes with a set of vscode or zed task to run each module independently.
Each module is run through [Air](https://github.com/air-verse/air) for hot reloading.
Run the task `Mir local dev` to start developing. For Zed, each task must be started individually as many tasks is not yet supported.
A set of tmux layouts can be found in the [tmux](./tmux) directory to run the modules if using tmux and tmuxifier.

Visit the `Justfile` to see the available commands and scripts to help develop locally.

### Cockpit

Cockpit is the Mir web interface, built with SvelteKit and located in `internal/ui/web/`. When developing the frontend, run the Vite dev server alongside `mir serve`:

```bash
# Terminal 1 — backend
mir serve

# Terminal 2 — frontend hot reload
cd internal/ui/web
npm install
npm run dev   # serves on http://localhost:5173
```

The Vite dev server origin must be allowed by the Cockpit CORS config. In `~/.config/mir/mir.yaml`, set:

```yaml
mir:
  cockpit:
    enabled: true
    allowedOrigins:
      - "http://localhost:5173"
```

The Cockpit frontend connects to NATS via WebSocket using `webTarget` from your active context (`~/.config/mir/cli.yaml`). The default local context points to `ws://localhost:9222` — no change needed for local dev.

Type-check the frontend with:

```bash
cd internal/ui/web
npm run check
```
