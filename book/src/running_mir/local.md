# Local Development & Testing

## Pre-requisites

To run Mir locally, you need to have the following installed:

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://golang.org/doc/install)
- [Rust](https://www.rust-lang.org/tools/install)  (To run Mir Book)
- [Make](https://www.gnu.org/software/make/) (Optional, for common tasks)

Once you have the above installed, you can run the following commands to complete installation:

```bash
# Linux
./scripts/tooling.sh

# Windows
go install github.com/air-verse/air@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/bufbuild/buf/cmd/buf@latest
cargo install mdbook
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

These services are defined in the [docker compose](./infra/dev/compose.yaml) file. To start the services, run the following command:

```bash
make docker-infra
# or
docker compose -f infra/dev/compose.yaml up --force-recreate
# or VsCode/Zed task
Mir infra dev
```

To build Mir binary, run the following command:

```bash
make build
# or
go build -o bin/mir cmds/mir/main.go
```

Mir binary come with a powerful CLI and TUI. It act as both the client and the server.
Once started as the server, open another terminal and you can use the CLI to interact with the system..
Use the `swarm` command to simulate a device connecting to the server to explore Mir ecosystem.

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

Tips (Linux), you can run `make mir-install` to install the binary to your system path.

To integrate your own device to the system, visit the [device tutorial](https://book.mirhub.io/using_mir/device_sdk.html).

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

Visit the `Makefile` to see the available commands and scripts to help develop locally.

Visit the [examples directory](./examples/) to see how to integrate your own device to the system [device example](./examples/telemetry_device/main.go) or build new modules.
