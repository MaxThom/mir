# Quick Start

To get started with Mir, you need to have the following installed:

- [Docker](https://www.docker.com/)

while installing, you can download the latest release of Mir from the [releases page](https://github.com/MaxThom/mir/releases).
From the download, extract the binary. Add it to your path for easier usage.

To begin, lets run the system:

```bash
# In one terminal, start a local Mir supporting infrastructure
mir infra up
# In another terminal, start the Mir server
mir serve
```

Voila! You now have a full Mir setup running. You have access to a configured Grafana ran by Mir infra:

```bash
# Title          <user>///<password>
# Grafana
localhost:3000 # admin///mir-operator
```

Find list of running services [here](running_mir/binary.md).

## Swarm

The CLI serves both client and server. Moreover, it provides a set of tools to
test and develop with the system. Swarm help mimic devices for testing or demo. Let's create
one virtual device:

```bash
# Open another terminal, and run
mir swarm --ids power
```

We now have a running virtual device with the device id `power`.

## Client

Open yet another terminal and run `mir device list`:

```
# Output
➜ mir dev ls
NAME/NAMESPACE                                DEVICE_ID        STATUS     LAST_HEARTHBEAT      LAST_SCHEMA_FETCH    LABELS
power/default                                 power            online     2024-11-21 02:39:09  2024-11-20 23:37:51
```

You should see your running device. To see its digital twin, use `mir device list power/default -o yaml`:

```yaml
# Output
apiVersion: v1alpha
apiName: device
meta:
    name: power
    namespace: default
    labels: {}
    annotations: {}
spec:
    deviceId: power
    disabled: false
properties: {}
status:
    online: true
    lastHearthbeat: 2024-11-21T02:40:49.312410397Z
    schema:
        packageNames:
            - google.protobuf
            - mir.device.v1
            - swarm.v1
        lastSchemaFetch: 2024-11-20T23:37:51.263010172Z
```

The digital twin is the virtual representation of the device. It contains the device's properties, status, schema and more.
Each device can be labeled with key value pairs for easy identification. Using those, you can query, send commands and manage subset of your devices.

Devices dont just exist, they communicate. Mir offers three methods of communication between the device and the server.
The first is telemetry, which is the device sending data to the server in a fire and forget manner.
The virtual device `power` we created earlier is sending some preconfigured telemetry.

To see device telemetry in action, run `mir telemetry power/default`:

```bash
# Output
➜ mir tlm ls power/default
1. power/default
swarm.v1.EnvironmentTlm{building=A, floor=1} localhost:3000/explore
swarm.v1.PowerConsuption{building=A, floor=2} localhost:3000/explore
```

Click on the link to see the telemetry in Grafana.

The second method of communication is commands. Commands are sent from the server to the device as a request-reply.

To send device commands, run `mir command send power/default` to see the available commands:

```bash
# Output
➜ mir cmd send power/default
1. power/default
swarm.v1.ChangeDataRateRequest{}
swarm.v1.ChangePowerRequest{}
```

To send a commands, use the following:

```bash
# See command payload
mir command send power/default -n swarm.v1.ChangePowerRequest -j
# Send command with modified payload
mir command send power/default -n swarm.v1.ChangePowerRequest -p '{"power": 5}'
# Quickly edit and send a command
mir command send power/default -n swarm.v1.ChangePowerRequest -e
```

```bash
# Output
➜ mir command send power/default -n swarm.v1.ChangePowerRequest -e
1. power/default COMMAND_RESPONSE_STATUS_SUCCESS
swarm.v1.ChangePowerResponse
{
  "success": true
}
```

Congratulations! You have a running Mir setup with a virtual device, telemetry, and commands.

## Next Steps

- To integrate your own device, visit the [DeviceSDK](integrating_mir/device/device_sdk.md)
- They are many more features to explore in the CLI, visit the [CLI](operating_mir/mir_cli_tui.md) for more commands and options.
