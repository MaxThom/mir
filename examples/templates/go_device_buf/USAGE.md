# Mir Device Template

Read full documentation on [Mir Book](book.mirhub.io)

## Install dev tools

The following are required to work with the DeviceSDK and Protofiles:

- [go](https://go.dev/doc/install): Go
- [protoc](https://protobuf.dev/installation/): Protocol buffer compiler.
- [buf](https://buf.build/docs/cli/installation/): Protocol buffer compiler.

After installing Go, you can install the rest via:

```bash
mir tools install
# or
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# or see documentation link above for manual install
```

## Generate the schema

```bash
make proto
# or
buf generate
buf lint
```

## Device Configuration

Device can be configured using, in order of priority:

1. SDK builder pattern
2. Environment variables (must be called from the builder)
3. Configuration file (must be called from the builder)

Caution, the DeviceID must be unique per instance.

It is possible to use the same configuration file for both Mir configuration and your project configuration:

```yaml
mir:
  target: "nats://127.0.0.1:4222" # Mir server URL
  logLevel: "info" # [trace|debug|info|warn|error|fatal]
  device:
    id: "example_device" # Must be unique per instance
  localStore:
    folderPath: ".store/" # Device storage location. Default to $HOME/.local/share/mir/mir.db
    retentionLimit: 160h # Retention of saved messages. Default to 1 week.
    persistenceType: "ifoffline" # [nostorage, ifoffline, always]
user:
  <your_data>
```

Add your data under user and edit `main.go` to add your config structure and update the builder:

```go
	ctx, cancel := context.WithCancel(context.Background())
	cfg := YourConfigStruct{}
	m, err := mir.Builder().
		DeviceId("example_device").
		Target("nats://127.0.0.1:4222").
		LogLevel(mir.LogLevelInfo).
		DefaultConfigFile().
		Schema(schemav1.File_schema_v1_schema_proto).
		BuildWithExtraConfig(&cfg)
```

## Run

```bash
make run
# or
go run cmd/main.go
```

## CLI

Use the Mir CLI to see device

```bash
# Device list
mir dev ls
# Telemetry list
mir dev tlm ls
# Command list
mir dev cmd ls
```

Yours to explore ! 🛰️

## Install binary to run on startup

### Systemd (Linux only)

create file `/etc/systemd/system/mir-dev.service`:
```bash
[Unit]
Description=Mir Device
After=network.target

[Service]
ExecStart=/home/mir/code/demo/bin/device
User=mir
Restart=always

[Install]
WantedBy=multi-user.target
```
Edit user and path accordinly

```bash
# Start the server automaticly on boot
sudo systemctl enable mir-dev.service
sudo systemctl disable mir-dev.service
# Start the service now
sudo systemctl start mir-dev.service
sudo systemctl stop mir-dev.service
# Status/Logs
sudo systemctl status mir-dev.service
journalctl -u mir-dev.service # -f (follow) -b (since boot)
```
