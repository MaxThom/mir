
# Device Telemetry Example

This example demonstrate how to use the Mir SDK:

- Connect to the system
- Send telemetry data
- Receive commands

To edit the schema, update `./proto/telemetry_device/v1` and run `just protogen` or `buf generate --clean --template examples/telemetry_device/buf.gen.yaml` from root directory.

Start the Mir server using the cli or tasks. Start this example using `go run main.go`.
