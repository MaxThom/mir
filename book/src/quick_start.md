# Quick Start

To get started with Mir, you need to have the following installed:

- [Docker](https://www.docker.com/)

while installing, you can download the latest release of Mir from the [releases page](https://github.com/MaxThom/mir/releases).
From the download, extract the binary. Add it to your path for easier usage.

You can also install the binary via Go (as it is a private repository, follow the [access guide](reference/access_mir.md)):
```bash
go install github.com/maxthom/mir/cmds/mir@latest
```

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

You should see your running device. To see its digital twin, use `mir device ls power/default`:

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
properties:
    desired:
        swarm.v1.ChangeDataRateProp:
            sec: 0
    reported:
        swarm.v1.ReportedProps:
            datarateSec: 0
            elevatorFloor: 0
status:
    online: true
    lastHearthbeat: 2025-02-13T12:27:46.295414516Z
    schema:
        packageNames:
            - google.protobuf
            - mir.device.v1
            - swarm.v1
        lastSchemaFetch: 2025-02-13T12:27:48.344371595Z
    properties:
        desired:
            swarm.v1.ChangeDataRateProp: 2025-02-13T12:27:48.349770386Z
        reported:
            swarm.v1.ReportedProps: 2025-02-13T12:27:48.353080302Z
```

The digital twin is the virtual representation of the device. It contains the device's properties, status, schema and more.
Each device can be labeled with key value pairs for easy identification. Using those, you can query, send commands and manage subset of your devices.

Use `mir dev -h` to see all available commands to create, update and delete devices.

## Communication

Devices dont just exist, they communicate. Mir offers three methods of communication between the device and the server.

### Telemetry

The first is telemetry, which is the device sending data to the server in a fire and forget manner.
The virtual device `power` we created earlier is sending some preconfigured telemetry.

To see device telemetry in action, run `mir tlm power/default`:

```bash
➜ mir tlm ls power/default
# Output
1. power/default
swarm.v1.EnvironmentTlm{building=A, floor=1} localhost:3000/explore
swarm.v1.PowerConsuption{building=A, floor=2} localhost:3000/explore
```

Click on the link to see the telemetry in Grafana.

### Commands

The second method of communication is commands. Commands are sent from the server to the device as a request-reply.

To send device commands, run `mir cmd send power/default` to see the available commands:

```bash
➜ mir cmd send power/default
# Output
1. power/default
swarm.v1.OpenDoorRequest{}
swarm.v1.SendElevatorRequest{}
```

To send a commands, use the following:

```bash
# See command payload
mir command send power/default -n swarm.v1.SendElevatorRequest -j
# Send command with modified payload
mir command send power/default -n swarm.v1.SendElevatorRequest -p '{"floor": 5}'
# Quickly edit and send a command
mir command send power/default -n swarm.v1.SendElevatorRequest -e
```

```bash
➜ mir command send power/default -n swarm.v1.SendElevatorRequest -e
# Output
1. power/default COMMAND_RESPONSE_STATUS_SUCCESS
swarm.v1.SendElevatorResponse
{
  "floor": 5
}
```

### Configuration

The third method of communication is configuration. Configuration is divided into desired properties and reported properties.
Contrary to commands, properties use an asynchronous messaging model and are written to storage.
They are meant to represent the desired and current state of the device.

To see the device configuration options, run `mir cfg send power/default`:

```bash
➜ mir config send power/default
# Output
1. power/default
swarm.v1.ChangeDataRateProp{}
```

To update configuration, use the following:

```bash
# See current config
mir config send power/default -n swarm.v1.ChangeDataRateProp -c
# See config template payload
mir config send power/default -n swarm.v1.ChangeDataRateProp -j
# Send config with modified payload
mir config send power/default -n swarm.v1.ChangeDataRateProp -p '{"sec": 5}'
# Quickly edit and send a config
mir config send power/default -n swarm.v1.ChangeDataRateProp -e
```

```bash
➜ mir config send power/default -n swarm.v1.ChangeDataRateProp -e
# Output
1. power/default
swarm.v1.ChangeDataRateProp{}
{
  "sec": 5
}
```

To see the updated configuration, run `mir dev ls power/default`:

```bash
apiVersion: v1alpha
apiName: device
meta:
    name: power
    namespace: default
...
properties:
    desired:
        swarm.v1.ChangeDataRateProp:
            sec: 5
    reported:
        swarm.v1.ReportedProps:
            datarateSec: 5
            elevatorFloor: 0
status:
...
    properties:
        desired:
            swarm.v1.ChangeDataRateProp: 2025-02-13T13:29:04.013967811Z
        reported:
            swarm.v1.ReportedProps: 2025-02-13T13:29:04.018232548Z
```

As you can see, the desired properties have been updated with our request and the device has reported the new values.

Under `Status.Properties`, you can see the timestamp of the desired and reported properties of their last update in UTC.

*! Desired properties can also be modified using the device api. Use `mir dev edit power/default` command.*

Congratulations! You have a running Mir setup with a virtual device, telemetry, commands and configuration!

## Next Steps

- To integrate your own device, visit the [DeviceSDK](integrating_mir/device/device_sdk.md)
- They are many more features to explore in the CLI, visit the [CLI](operating_mir/mir_cli_tui.md) for more commands and options.
