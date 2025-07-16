# 🔄 Commands

The command path enables synchronous, request-response communication for device control.

## Characteristics

- **Request-response**: Every command expects a reply
- **Synchronous**: Caller waits for execution
- **Timeout handling**: Configurable command timeouts
- **Audit trail**: All commands are logged

## How It Works

```
CLI/API                   Mir                     Device
  │                        │                         │
  ├── Send Command ───────▶│                         │
  │                        ├── Validate Permissions  │
  │                        ├── Route to Device ─────▶│
  │                        │                         ├── Execute
  │                        │◀── Command Response ────┤
  │◀── Return Response ────┤                         │
  │                        ├── Log to EventStore     │
```

## Implementation

### Schema Definition

```protobuf
message ActivateRelayCmd {
  option (mir.device.v1.message_type) = MESSAGE_TYPE_TELECOMMAND;

    int32 relay_id = 1;
    int32 duration = 2;  // seconds
}

message ActivateRelayResp {
    bool success = 1;
    string message = 2;
}
```

### Device

```go
// Register command handler
device.HandleCommand(
		&schemav1.ActivateRelayCmd{},
		func(msg proto.Message) (proto.Message, error) {
			cmd := msg.(*schemav1.ActivateRelayCmd)

			// Process command ...
			err := hardware.ActivateRelay(cmd.RelayId, cmd.Duration)
      if err != nil {
        return nil, err
      }

			return &schemav1.ActivateRelayResp{
				Success: true,
				Message: fmt.Sprintf("Relay %d activated for %d seconds", cmd.RelayId, cmd.Duration),
			}, nil
		},
	)
```

### Sending Commands

Using the CLI:

```bash
mir dev cmd send <name>/<namespace> -n ActivateRelayCmd -p '{"relay_id": 1, "duration": 60}'

# Response
{
  "success": true,
  "message": "Relay 1 activated for 60 seconds"
}
```

## Use Cases

- **Actuator control**: Turn on/off lights, motors, valves
- **Device queries**: Get current status, diagnostics
- **Configuration changes**: Update settings immediately
- **Firmware operations**: Trigger updates, reboots
- **Maintenance**: Run diagnostics, calibration

## Best Practices

1. **Validate inputs**: Check parameters before execution
2. **Handle timeouts**: Implement proper timeout logic
3. **Return meaningful errors**: Help debugging issues
4. **Keep it fast**: Commands should execute quickly
5. **Idempotent design**: Safe to retry if needed
