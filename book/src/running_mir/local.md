# Local Development & Testing

## Pre-requisites

To run Mir locally, you need to have the following installed:

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://golang.org/doc/install)
- [Rust](https://www.rust-lang.org/tools/install)  (To run Mir Book)
- [Just](https://github.com/casey/just) (Optional, for common tasks)
- [Protoc](https://grpc.io/docs/protoc-installation/)

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

These services are defined in the [docker compose](./infra/local_support/compose.yaml) file. To start the services, run the following command:

```bash
just infra
# or
docker compose -f infra/local_support/compose.yaml up --force-recreate
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
mir device
# to visualize the telemetry
mir telemetry
# to send command to devices
mir command
# to explore and upload schemas
mir schema
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
