# Mir Device Template

## Install dev tools

- [protoc](https://grpc.io/docs/protoc-installation/): Protocol buffer compiler.

It must be manually installed via your package manager:

```bash
# Debian, Ubuntu, Raspian
sudo apt install protobuf-compiler
# Arch based
sudo pacman -S protobuf
```

The following can be installed via `go install` or using Mir CLI:

```bash
mir tools install
```

## Generate the schema

```bash
make proto
# or
protoc \
	--proto_path=schemav1/ \
    --go_out=schemav1 \
    --go_opt=paths=source_relative \
    schemav1/schema.proto
```

Add schema import to your `cmd/main.go`
```go
	schemav1 "github.com/<path>/schema/v1"
```

Uncomment line 17, edit deviceid and edit Target url if not local:

```go
m, err := mir.Builder().
	DeviceId("template").
	Target("nats://192.168.3.73:4222").
	Schema(schemav1.File_schema_proto).
	Build()
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
mir tlm ls
# Command list
mir cmd ls
```

Yours to explore ! 🛰️

## Systemd (Linux only)

To install binary so it run on startup:

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
